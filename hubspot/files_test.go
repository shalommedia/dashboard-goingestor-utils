package hubspot

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestUploadPDFToFolder_BuildsMultipartAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusCreated,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "122692044085",
				"name": "proposal.pdf",
				"path": "/library/proposals/proposal.pdf",
				"access": "PRIVATE"
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.UploadPDFToFolder(context.Background(), UploadPDFToFolderRequest{
		FileName: "proposal.pdf",
		FileData: []byte("%PDF-1.7 test content"),
		FolderID: "122692510820",
	})
	if err != nil {
		t.Fatalf("UploadPDFToFolder returned error: %v", err)
	}

	if resp.ID != "122692044085" || resp.Name != "proposal.pdf" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/files/v3/files" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}

	if got := req.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data; boundary=") {
		t.Fatalf("expected multipart content-type, got %q", got)
	}

	if err := req.ParseMultipartForm(1024 * 1024); err != nil {
		t.Fatalf("parse multipart form: %v", err)
	}

	if got := req.FormValue("folderId"); got != "122692510820" {
		t.Fatalf("unexpected folderId field: %q", got)
	}

	if got := req.FormValue("fileName"); got != "proposal.pdf" {
		t.Fatalf("unexpected fileName field: %q", got)
	}

	if got := req.FormValue("options"); got != `{"access":"PRIVATE"}` {
		t.Fatalf("unexpected options field: %q", got)
	}

	uploadedFile, header, err := req.FormFile("file")
	if err != nil {
		t.Fatalf("read uploaded file part: %v", err)
	}
	defer uploadedFile.Close()

	if header.Filename != "proposal.pdf" {
		t.Fatalf("unexpected multipart filename: %q", header.Filename)
	}

	fileData, err := io.ReadAll(uploadedFile)
	if err != nil {
		t.Fatalf("read multipart file data: %v", err)
	}

	if string(fileData) != "%PDF-1.7 test content" {
		t.Fatalf("unexpected multipart file content: %q", string(fileData))
	}
}

func TestUploadPDFToFolder_RequiresPDFExtension(t *testing.T) {
	t.Parallel()

	client, err := New(Config{Token: "token-123", Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.UploadPDFToFolder(context.Background(), UploadPDFToFolderRequest{
		FileName: "proposal.txt",
		FileData: []byte("test"),
		FolderID: "123",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "file name must end with .pdf") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadPDFToFolder_RequiresFolderID(t *testing.T) {
	t.Parallel()

	client, err := New(Config{Token: "token-123", Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.UploadPDFToFolder(context.Background(), UploadPDFToFolderRequest{
		FileName: "proposal.pdf",
		FileData: []byte("%PDF"),
		FolderID: " ",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "folder id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
