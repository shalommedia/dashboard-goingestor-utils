package s3client

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type mockPutObjectClient struct {
	lastInput *s3.PutObjectInput
	err       error
}

type mockGetObjectClient struct {
	output *s3.GetObjectOutput
	err    error
}

func (m *mockGetObjectClient) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.output, nil
}

func (m *mockPutObjectClient) PutObject(_ context.Context, params *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	m.lastInput = params
	if m.err != nil {
		return nil, m.err
	}

	return &s3.PutObjectOutput{}, nil
}

func TestPutObjectStream_SetsContentLengthWhenKnown(t *testing.T) {
	t.Parallel()

	client := &mockPutObjectClient{}
	body := strings.NewReader("hello")

	err := putObjectStream(context.Background(), client, "bucket-a", "contacts.csv", 5, "text/csv", body)
	if err != nil {
		t.Fatalf("putObjectStream returned error: %v", err)
	}

	if client.lastInput == nil {
		t.Fatal("expected PutObject input to be captured")
	}

	if client.lastInput.Bucket == nil || *client.lastInput.Bucket != "bucket-a" {
		t.Fatalf("unexpected bucket: %#v", client.lastInput.Bucket)
	}

	if client.lastInput.Key == nil || *client.lastInput.Key != "contacts.csv" {
		t.Fatalf("unexpected key: %#v", client.lastInput.Key)
	}

	if client.lastInput.ContentType == nil || *client.lastInput.ContentType != "text/csv" {
		t.Fatalf("unexpected content type: %#v", client.lastInput.ContentType)
	}

	if client.lastInput.ContentLength == nil || *client.lastInput.ContentLength != 5 {
		t.Fatalf("unexpected content length: %#v", client.lastInput.ContentLength)
	}

	if client.lastInput.Body == nil {
		t.Fatal("expected body to be forwarded")
	}
}

func TestPutObjectStream_OmitsContentLengthWhenUnknown(t *testing.T) {
	t.Parallel()

	client := &mockPutObjectClient{}

	err := putObjectStream(context.Background(), client, "bucket-a", "contacts.csv", -1, "text/csv", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("putObjectStream returned error: %v", err)
	}

	if client.lastInput == nil {
		t.Fatal("expected PutObject input to be captured")
	}

	if client.lastInput.ContentLength != nil {
		t.Fatalf("expected content length to be nil, got %#v", *client.lastInput.ContentLength)
	}
}

func TestPutObjectStream_WrapsS3Error(t *testing.T) {
	t.Parallel()

	sentinelErr := errors.New("s3 unavailable")
	client := &mockPutObjectClient{err: sentinelErr}

	err := putObjectStream(context.Background(), client, "bucket-b", "object.txt", 11, "text/plain", strings.NewReader("hello world"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, sentinelErr) {
		t.Fatalf("expected wrapped s3 error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "bucket=bucket-b") {
		t.Fatalf("expected bucket context in error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "key=object.txt") {
		t.Fatalf("expected key context in error, got: %v", err)
	}
}

func TestPutObject_UsesInMemoryBytesReader(t *testing.T) {
	t.Parallel()

	client := &mockPutObjectClient{}
	input := UploadInput{
		Bucket:      "bucket-c",
		Key:         "payload.bin",
		Body:        []byte("abc123"),
		ContentType: "application/octet-stream",
	}

	err := putObject(context.Background(), client, input)
	if err != nil {
		t.Fatalf("putObject returned error: %v", err)
	}

	if client.lastInput == nil || client.lastInput.Body == nil {
		t.Fatal("expected body to be set")
	}

	body, readErr := io.ReadAll(client.lastInput.Body)
	if readErr != nil {
		t.Fatalf("failed to read forwarded body: %v", readErr)
	}

	if string(body) != "abc123" {
		t.Fatalf("unexpected body content: %q", string(body))
	}
}

func TestPutObjectStream_PropagatesCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &mockPutObjectClient{
		err: context.Canceled,
	}

	err := putObjectStream(ctx, client, "bucket-z", "canceled.txt", -1, "text/plain", strings.NewReader("data"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "bucket=bucket-z") {
		t.Fatalf("expected bucket context in error, got: %v", err)
	}
}

func TestGetObjectWithLimit_FailsWhenContentLengthExceedsLimit(t *testing.T) {
	t.Parallel()

	contentLength := int64(10)
	client := &mockGetObjectClient{
		output: &s3.GetObjectOutput{
			Body:          io.NopCloser(strings.NewReader("abcdefghij")),
			ContentLength: &contentLength,
		},
	}

	_, err := getObjectWithLimit(context.Background(), client, "bucket-a", "large.txt", 5)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds max read size") {
		t.Fatalf("expected max size error, got: %v", err)
	}
}

func TestGetObjectWithLimit_FailsWhenStreamExceedsLimit(t *testing.T) {
	t.Parallel()

	contentLength := int64(-1)
	client := &mockGetObjectClient{
		output: &s3.GetObjectOutput{
			Body:          io.NopCloser(strings.NewReader("0123456789")),
			ContentLength: &contentLength,
		},
	}

	_, err := getObjectWithLimit(context.Background(), client, "bucket-a", "stream.txt", 5)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds max read size") {
		t.Fatalf("expected max size error, got: %v", err)
	}
}

func TestGetObjectWithLimit_WrapsCanceledReadError(t *testing.T) {
	t.Parallel()

	readErr := context.Canceled
	client := &mockGetObjectClient{
		output: &s3.GetObjectOutput{
			Body: &errorReadCloser{err: readErr},
		},
	}

	_, err := getObjectWithLimit(context.Background(), client, "bucket-a", "cancel.txt", 1024)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected wrapped canceled error, got: %v", err)
	}
}

type errorReadCloser struct {
	err error
}

func (e *errorReadCloser) Read(_ []byte) (int, error) {
	return 0, e.err
}

func (e *errorReadCloser) Close() error {
	return nil
}

func TestGetObject_WrapsGetObjectError(t *testing.T) {
	t.Parallel()

	sentinelErr := errors.New("network issue")
	client := &mockGetObjectClient{err: sentinelErr}

	_, err := getObject(context.Background(), client, "bucket-a", "file.txt")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, sentinelErr) {
		t.Fatalf("expected wrapped get object error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "bucket=bucket-a") {
		t.Fatalf("expected bucket context in error, got: %v", err)
	}
}
