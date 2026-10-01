package sendly

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func multerFilterServer(t *testing.T, allowed []string, refusal, reply string, seen *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		files := r.MultipartForm.File["file"]
		if len(files) != 1 {
			t.Errorf("expected one file part, got %d", len(files))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		contentType := files[0].Header.Get("Content-Type")
		*seen = append(*seen, files[0].Filename+" "+contentType)
		for _, a := range allowed {
			if contentType == a {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(reply))
				return
			}
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": refusal})
	}))
}

func TestMediaUpload_SendsTheFileContentType(t *testing.T) {
	var seen []string
	server := multerFilterServer(t, []string{"image/jpeg", "image/png", "image/gif"},
		"Only JPEG, PNG, and GIF images are allowed for MMS",
		`{"id":"file_1","url":"https://media.example.com/file_1.png","contentType":"image/png","sizeBytes":12}`, &seen)
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 16)...)
	file, err := client.Media.Upload(context.Background(), "photo.png", bytes.NewReader(png))
	if err != nil {
		t.Fatalf("unexpected error for a PNG: %v", err)
	}
	if file.ID != "file_1" || file.ContentType != "image/png" {
		t.Errorf("unexpected media file: %+v", file)
	}

	jpeg := append([]byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00"), make([]byte, 16)...)
	if _, err := client.Media.Upload(context.Background(), "photo.jpg", bytes.NewReader(jpeg)); err != nil {
		t.Fatalf("unexpected error for a JPEG: %v", err)
	}

	want := []string{"photo.png image/png", "photo.jpg image/jpeg"}
	if len(seen) != len(want) || seen[0] != want[0] || seen[1] != want[1] {
		t.Errorf("expected parts %v, got %v", want, seen)
	}
}

func TestFileContentType(t *testing.T) {
	tests := []struct {
		filename string
		data     []byte
		want     string
	}{
		{"photo.png", []byte("\x89PNG\r\n\x1a\n0000"), "image/png"},
		{"photo.gif", []byte("GIF89a0000"), "image/gif"},
		{"ein.pdf", []byte("%PDF-1.7\n"), "application/pdf"},
		{"scan.jpg", []byte("not really an image"), "image/jpeg"},
		{"photo.webp", []byte{0x00, 0x01, 0x02}, "image/webp"},
		{"blob", []byte{0x00, 0x01, 0x02}, "application/octet-stream"},
	}
	for _, tt := range tests {
		if got := fileContentType(tt.filename, tt.data); got != tt.want {
			t.Errorf("fileContentType(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}

func TestCreateFilePart_KeepsTheFilename(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := createFilePart(writer, "file", `my "best" photo.png`, []byte("\x89PNG\r\n\x1a\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	part.Write([]byte("\x89PNG\r\n\x1a\n"))
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())
	p, err := reader.NextPart()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.FormName() != "file" || p.FileName() != `my "best" photo.png` {
		t.Errorf("expected field 'file' and the original filename, got %q and %q", p.FormName(), p.FileName())
	}
	if ct := p.Header.Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected Content-Type image/png, got %q", ct)
	}
}
