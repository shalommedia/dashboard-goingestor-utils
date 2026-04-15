package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
)

const (
	defaultFileAccess = "PRIVATE"
)

// UploadPDFToFolderRequest controls the file upload payload for HubSpot Files API.
type UploadPDFToFolderRequest struct {
	FileName string
	FileData []byte
	FolderID string
	Access   string
}

// UploadedFile is the minimal response model from HubSpot Files API upload endpoint.
type UploadedFile struct {
	ID                string `json:"id"`
	Name              string `json:"name,omitempty"`
	Path              string `json:"path,omitempty"`
	URL               string `json:"url,omitempty"`
	DefaultHostingURL string `json:"defaultHostingUrl,omitempty"`
	Access            string `json:"access,omitempty"`
}

// UploadPDFToFolder uploads a PDF file to HubSpot Files in a target folder id.
func (c *Client) UploadPDFToFolder(ctx context.Context, req UploadPDFToFolderRequest) (UploadedFile, error) {
	trimmedFileName := strings.TrimSpace(req.FileName)
	if trimmedFileName == "" {
		return UploadedFile{}, errors.New("file name is required")
	}

	if !strings.EqualFold(fileExtension(trimmedFileName), ".pdf") {
		return UploadedFile{}, errors.New("file name must end with .pdf")
	}

	if len(req.FileData) == 0 {
		return UploadedFile{}, errors.New("file data is required")
	}

	trimmedFolderID := strings.TrimSpace(req.FolderID)
	if trimmedFolderID == "" {
		return UploadedFile{}, errors.New("folder id is required")
	}

	access := strings.TrimSpace(req.Access)
	if access == "" {
		access = defaultFileAccess
	}

	body, contentType, err := buildPDFUploadBody(trimmedFileName, req.FileData, trimmedFolderID, access)
	if err != nil {
		return UploadedFile{}, fmt.Errorf("build pdf upload request file=%s folderId=%s: %w", trimmedFileName, trimmedFolderID, err)
	}

	resp, err := c.Do(ctx, http.MethodPost, "/files/v3/files", &body, map[string]string{
		"Content-Type": contentType,
	})
	if err != nil {
		return UploadedFile{}, fmt.Errorf("upload pdf file=%s folderId=%s: %w", trimmedFileName, trimmedFolderID, err)
	}
	defer resp.Body.Close()

	var parsed UploadedFile
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return UploadedFile{}, fmt.Errorf("decode upload pdf response file=%s folderId=%s: %w", trimmedFileName, trimmedFolderID, err)
	}

	return parsed, nil
}

func buildPDFUploadBody(fileName string, fileData []byte, folderID, access string) (bytes.Buffer, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	fileHeader := make(textproto.MIMEHeader)
	fileHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName))
	fileHeader.Set("Content-Type", "application/pdf")

	filePart, err := writer.CreatePart(fileHeader)
	if err != nil {
		return bytes.Buffer{}, "", fmt.Errorf("create file part: %w", err)
	}

	if _, err := filePart.Write(fileData); err != nil {
		return bytes.Buffer{}, "", fmt.Errorf("write file part: %w", err)
	}

	if err := writer.WriteField("folderId", folderID); err != nil {
		return bytes.Buffer{}, "", fmt.Errorf("write folder id field: %w", err)
	}

	if err := writer.WriteField("fileName", fileName); err != nil {
		return bytes.Buffer{}, "", fmt.Errorf("write file name field: %w", err)
	}

	if err := writer.WriteField("options", fmt.Sprintf(`{"access":"%s"}`, access)); err != nil {
		return bytes.Buffer{}, "", fmt.Errorf("write options field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return bytes.Buffer{}, "", fmt.Errorf("close multipart writer: %w", err)
	}

	return body, writer.FormDataContentType(), nil
}

func fileExtension(fileName string) string {
	lastDot := strings.LastIndex(fileName, ".")
	if lastDot < 0 {
		return ""
	}

	return fileName[lastDot:]
}
