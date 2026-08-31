package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestTemplatesPreview_DeprecatedFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/templates/tpl_1/preview" {
			t.Errorf("expected path '/templates/tpl_1/preview', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"template_id": "tpl_1",
			"original_text": "Hi {{name}}",
			"rendered_text": "Hi Ada",
			"character_count": 6,
			"segment_count": 1
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	preview, err := client.Templates.Preview(context.Background(), "tpl_1", map[string]string{"name": "Ada"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if preview.TemplateID != "tpl_1" {
		t.Errorf("expected TemplateID 'tpl_1', got '%s'", preview.TemplateID)
	}
	if preview.RenderedText != "Hi Ada" {
		t.Errorf("expected RenderedText 'Hi Ada', got '%s'", preview.RenderedText)
	}
	if preview.CharacterCount != 6 || preview.SegmentCount != 1 {
		t.Errorf("expected counts to decode, got %d/%d", preview.CharacterCount, preview.SegmentCount)
	}
	if preview.ID != "tpl_1" {
		t.Errorf("expected deprecated ID to mirror TemplateID, got '%s'", preview.ID)
	}
	if preview.PreviewText != "Hi Ada" {
		t.Errorf("expected deprecated PreviewText to mirror RenderedText, got '%s'", preview.PreviewText)
	}
	if preview.Name != "" || len(preview.Variables) != 0 {
		t.Errorf("expected deprecated Name and Variables to stay empty, got '%s'/%+v", preview.Name, preview.Variables)
	}
}

func TestTemplatesPreview_LegacyPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "tpl_1",
			"name": "Greeting",
			"preview_text": "Hi Ada",
			"variables": [{"key": "name", "type": "string"}]
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	preview, err := client.Templates.Preview(context.Background(), "tpl_1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if preview.ID != "tpl_1" {
		t.Errorf("expected ID 'tpl_1' to survive a payload without template_id, got '%s'", preview.ID)
	}
	if preview.PreviewText != "Hi Ada" {
		t.Errorf("expected PreviewText 'Hi Ada' to survive a payload without rendered_text, got '%s'", preview.PreviewText)
	}
	if preview.Name != "Greeting" {
		t.Errorf("expected Name 'Greeting', got '%s'", preview.Name)
	}
	if len(preview.Variables) != 1 || preview.Variables[0].Key != "name" {
		t.Errorf("expected Variables to decode, got %+v", preview.Variables)
	}
}

func TestTemplatePreview_RoundTrip(t *testing.T) {
	original := TemplatePreview{
		TemplateID:     "tpl_1",
		OriginalText:   "Hi {{name}}",
		RenderedText:   "Hi Ada",
		CharacterCount: 6,
		SegmentCount:   1,
		ID:             "tpl_1",
		Name:           "Greeting",
		PreviewText:    "Hi Ada",
		Variables:      []TemplateVariable{{Key: "name", Type: "string", Fallback: "there"}},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded TemplatePreview
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip changed the value:\n before %+v\n after  %+v", original, decoded)
	}
}

func TestTemplatesClone_RouteNotAvailable(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{
			Code:    "NOT_FOUND",
			Message: "Route not found",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	_, err := client.Templates.Clone(context.Background(), "tpl_1", &CloneTemplateRequest{Name: "Copy"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
	if path != "/templates/tpl_1/clone" {
		t.Errorf("expected path '/templates/tpl_1/clone', got '%s'", path)
	}
}
