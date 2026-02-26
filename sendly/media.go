package sendly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
)

// MediaService handles media upload operations.
type MediaService struct {
	client *Client
}

// Upload uploads a media file for use in MMS messages.
func (s *MediaService) Upload(ctx context.Context, filename string, file io.Reader) (*MediaFile, error) {
	if file == nil {
		return nil, &ValidationError{APIError: APIError{Message: "file is required"}}
	}
	if filename == "" {
		return nil, &ValidationError{APIError: APIError{Message: "filename is required"}}
	}

	if err := s.client.rateLimiter.Wait(ctx); err != nil {
		return nil, &NetworkError{Message: "rate limiter error", Err: err}
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return nil, &NetworkError{Message: "failed to create form file", Err: err}
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, &NetworkError{Message: "failed to write file data", Err: err}
	}

	if err := writer.Close(); err != nil {
		return nil, &NetworkError{Message: "failed to close multipart writer", Err: err}
	}

	fullURL := s.client.BaseURL + "/media"

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, &buf)
	if err != nil {
		return nil, &NetworkError{Message: "failed to create request", Err: err}
	}

	req.Header.Set("Authorization", "Bearer "+s.client.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sendly-go/"+Version)

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Message: "request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &NetworkError{Message: "failed to read response body", Err: err}
	}

	if resp.StatusCode >= 400 {
		return nil, s.client.handleErrorResponse(resp, respBody)
	}

	var result MediaFile
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, &NetworkError{Message: "failed to unmarshal response", Err: err}
	}

	return &result, nil
}

// Delete deletes an uploaded media file by ID.
func (s *MediaService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return &ValidationError{APIError: APIError{Message: "media ID is required"}}
	}

	return s.client.request(ctx, "DELETE", fmt.Sprintf("/media/%s", id), nil, nil)
}
