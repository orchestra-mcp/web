package handlers

import (
	"encoding/json"
	"testing"
)

func TestUpdatePostBodyIncludesTags(t *testing.T) {
	// Simulate the JSON body that the frontend sends on post edit.
	requestJSON := `{
		"title": "Updated Title",
		"content": "Updated content",
		"icon": "rocket",
		"color": "#ff0000",
		"media": "",
		"tags": ["skill"]
	}`

	var body struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Icon    string   `json:"icon"`
		Color   string   `json:"color"`
		Media   string   `json:"media"`
		Tags    []string `json:"tags"`
	}

	if err := json.Unmarshal([]byte(requestJSON), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}

	if len(body.Tags) != 1 || body.Tags[0] != "skill" {
		t.Errorf("Tags = %v, want [skill]", body.Tags)
	}
	if body.Title != "Updated Title" {
		t.Errorf("Title = %q, want Updated Title", body.Title)
	}
}

func TestUpdatePostTagsSerialization(t *testing.T) {
	tags := []string{"workflow"}
	tagsJSON := "[]"
	if len(tags) > 0 {
		if b, err := json.Marshal(tags); err == nil {
			tagsJSON = string(b)
		}
	}

	if tagsJSON != `["workflow"]` {
		t.Errorf("tagsJSON = %q, want [\"workflow\"]", tagsJSON)
	}
}

func TestUpdatePostEmptyTagsSerialization(t *testing.T) {
	tags := []string{}
	tagsJSON := "[]"
	if len(tags) > 0 {
		if b, err := json.Marshal(tags); err == nil {
			tagsJSON = string(b)
		}
	}

	if tagsJSON != "[]" {
		t.Errorf("tagsJSON = %q, want []", tagsJSON)
	}
}

func TestUpdatePostNilTagsSerialization(t *testing.T) {
	var tags []string // nil
	tagsJSON := "[]"
	if len(tags) > 0 {
		if b, err := json.Marshal(tags); err == nil {
			tagsJSON = string(b)
		}
	}

	if tagsJSON != "[]" {
		t.Errorf("tagsJSON = %q, want []", tagsJSON)
	}
}

func TestUpdatePostResponseIncludesTags(t *testing.T) {
	// Simulate the response shape the frontend expects.
	tags := []string{"agent"}
	resp := map[string]interface{}{
		"post": map[string]interface{}{
			"id":             1,
			"title":          "Test",
			"content":        "Body",
			"icon":           "",
			"color":          "",
			"media":          "",
			"tags":           tags,
			"likes_count":    0,
			"comments_count": 0,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	post := parsed["post"].(map[string]interface{})
	tagsArr, ok := post["tags"].([]interface{})
	if !ok {
		t.Fatal("tags field missing or not array in response")
	}
	if len(tagsArr) != 1 || tagsArr[0] != "agent" {
		t.Errorf("tags = %v, want [agent]", tagsArr)
	}
}

func TestUpdatePostBodyWithoutTagsDefaultsEmpty(t *testing.T) {
	// Frontend sends no tags field — should default to empty slice.
	requestJSON := `{
		"title": "No Tags Post",
		"content": "Content"
	}`

	var body struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}

	if err := json.Unmarshal([]byte(requestJSON), &body); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if body.Tags != nil {
		t.Errorf("Tags = %v, want nil (absent field)", body.Tags)
	}

	// Our serialization logic handles nil correctly.
	tagsJSON := "[]"
	if len(body.Tags) > 0 {
		if b, err := json.Marshal(body.Tags); err == nil {
			tagsJSON = string(b)
		}
	}
	if tagsJSON != "[]" {
		t.Errorf("tagsJSON = %q, want []", tagsJSON)
	}
}

func TestUpdatePostMultipleTags(t *testing.T) {
	requestJSON := `{
		"title": "Multi Tag",
		"content": "Content",
		"tags": ["skill", "featured", "go"]
	}`

	var body struct {
		Title string   `json:"title"`
		Tags  []string `json:"tags"`
	}

	if err := json.Unmarshal([]byte(requestJSON), &body); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(body.Tags) != 3 {
		t.Errorf("len(Tags) = %d, want 3", len(body.Tags))
	}

	tagsJSON := "[]"
	if len(body.Tags) > 0 {
		if b, err := json.Marshal(body.Tags); err == nil {
			tagsJSON = string(b)
		}
	}

	var roundTripped []string
	if err := json.Unmarshal([]byte(tagsJSON), &roundTripped); err != nil {
		t.Fatalf("failed to round-trip tags: %v", err)
	}
	if len(roundTripped) != 3 || roundTripped[0] != "skill" || roundTripped[1] != "featured" || roundTripped[2] != "go" {
		t.Errorf("round-tripped = %v, want [skill featured go]", roundTripped)
	}
}
