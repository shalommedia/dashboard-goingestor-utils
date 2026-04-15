package s3client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	defaultClient *s3.Client
	defaultErr    error
	once          sync.Once
)

// putObjectAPI captures the subset of the AWS SDK used by putObject.
type putObjectAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// getObjectAPI captures the subset of the AWS SDK used by getObject.
type getObjectAPI interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// New creates an S3 client from the ambient AWS configuration.
func New(ctx context.Context) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg), nil
}

// Default returns a lazily initialized shared client for reuse across Lambda invocations.
func Default(ctx context.Context) (*s3.Client, error) {
	once.Do(func() {
		defaultClient, defaultErr = New(ctx)
	})

	return defaultClient, defaultErr
}

// PutObject uploads an in-memory payload to S3 using the shared client.
func PutObject(ctx context.Context, input UploadInput) error {
	client, err := Default(ctx)
	if err != nil {
		return fmt.Errorf("create s3 client: %w", err)
	}

	return putObject(ctx, client, input)
}

// GetObject downloads an object from S3 and returns its full contents in memory.
func GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	client, err := Default(ctx)
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	return getObjectWithLimit(ctx, client, bucket, key, 0)
}

// GetObjectWithLimit downloads an object from S3 and enforces an upper bound on bytes read.
// Set maxBytes to 0 to disable limits.
func GetObjectWithLimit(ctx context.Context, bucket, key string, maxBytes int64) ([]byte, error) {
	client, err := Default(ctx)
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	return getObjectWithLimit(ctx, client, bucket, key, maxBytes)
}

// PutObjectStream uploads a stream to S3 without buffering the entire payload in memory.
// The contentLength parameter should be the exact number of bytes that will be written; use -1 if unknown.
// This is ideal for large objects or streaming data sources (e.g., from HubSpot API pagination).
func PutObjectStream(ctx context.Context, bucket, key string, contentLength int64, contentType string, body io.Reader) error {
	client, err := Default(ctx)
	if err != nil {
		return fmt.Errorf("create s3 client: %w", err)
	}

	return putObjectStream(ctx, client, bucket, key, contentLength, contentType, body)
}

// putObject builds the SDK request and executes the S3 upload.
func putObject(ctx context.Context, client putObjectAPI, input UploadInput) error {
	// S3 expects the request body as an io.Reader, so the byte slice is wrapped in a reader here.
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &input.Bucket,
		Key:         &input.Key,
		Body:        bytes.NewReader(input.Body),
		ContentType: &input.ContentType,
	})
	if err != nil {
		return fmt.Errorf("put object to s3 bucket=%s key=%s: %w", input.Bucket, input.Key, err)
	}

	return nil
}

// putObjectStream executes the streaming S3 upload.
func putObjectStream(ctx context.Context, client putObjectAPI, bucket, key string, contentLength int64, contentType string, body io.Reader) error {
	input := &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        body,
		ContentType: &contentType,
	}

	// If content length is known, set it for improved S3 handling.
	if contentLength > 0 {
		input.ContentLength = &contentLength
	}

	_, err := client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("put object stream to s3 bucket=%s key=%s: %w", bucket, key, err)
	}

	return nil
}

// getObject downloads the object body and reads it fully into memory.
func getObject(ctx context.Context, client getObjectAPI, bucket, key string) ([]byte, error) {
	return getObjectWithLimit(ctx, client, bucket, key, 0)
}

// getObjectWithLimit downloads the object body and enforces a maximum read size when configured.
func getObjectWithLimit(ctx context.Context, client getObjectAPI, bucket, key string, maxBytes int64) ([]byte, error) {
	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("get object from s3 bucket=%s key=%s: %w", bucket, key, err)
	}
	defer output.Body.Close()

	if maxBytes > 0 && output.ContentLength != nil && *output.ContentLength > maxBytes {
		return nil, fmt.Errorf("object exceeds max read size bucket=%s key=%s max_bytes=%d content_length=%d", bucket, key, maxBytes, *output.ContentLength)
	}

	reader := io.Reader(output.Body)
	if maxBytes > 0 {
		reader = io.LimitReader(output.Body, maxBytes+1)
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read object body from s3 bucket=%s key=%s: %w", bucket, key, err)
	}

	if maxBytes > 0 && int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("object exceeds max read size bucket=%s key=%s max_bytes=%d", bucket, key, maxBytes)
	}

	return body, nil
}
