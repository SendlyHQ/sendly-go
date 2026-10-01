package sendly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strings"
)

// MediaService handles media upload operations.
type MediaService struct {
	client *Client
}

// Upload uploads a JPEG, PNG or GIF image (up to 600 KB) for use in MMS
// messages. The file is labelled with the type its content shows, or failing
// that the type its filename's extension names. Any other type, or a larger
// file, is refused with a *SendlyError (HTTP 500); a file whose content is not
// the image type it is labelled with is refused with a *ValidationError (code
// invalid_file).
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

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, &NetworkError{Message: "failed to write file data", Err: err}
	}

	part, err := createFilePart(writer, "file", filepath.Base(filename), data)
	if err != nil {
		return nil, &NetworkError{Message: "failed to create form file", Err: err}
	}

	if _, err := part.Write(data); err != nil {
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
	req.Header.Set("Idempotency-Key", generateIdempotencyKey())

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

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func createFilePart(w *multipart.Writer, fieldName, filename string, data []byte) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
		quoteEscaper.Replace(fieldName), quoteEscaper.Replace(filename)))
	h.Set("Content-Type", fileContentType(filename, data))
	return w.CreatePart(h)
}

func fileContentType(filename string, data []byte) string {
	sniffed := http.DetectContentType(data)
	if sniffed != "application/octet-stream" && !strings.HasPrefix(sniffed, "text/plain") {
		return sniffed
	}
	if byExtension := mime.TypeByExtension(filepath.Ext(filename)); byExtension != "" {
		return byExtension
	}
	return "application/octet-stream"
}

// Delete deletes an uploaded media file by ID.
func (s *MediaService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return &ValidationError{APIError: APIError{Message: "media ID is required"}}
	}

	return s.client.request(ctx, "DELETE", fmt.Sprintf("/media/%s", url.PathEscape(id)), nil, nil)
}
