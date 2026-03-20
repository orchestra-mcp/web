package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestFetchOg(t *testing.T) {
	// Serve a test HTML page with OG tags
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<meta property="og:title" content="Test Page Title" />
<meta property="og:description" content="A test description for the page" />
<meta property="og:image" content="https://example.com/image.png" />
<meta property="og:site_name" content="Example Site" />
<title>Fallback Title</title>
</head>
<body><p>Hello</p></body>
</html>`))
	}))
	defer ts.Close()

	h := NewOgPreviewHandler()

	mustParse := func(raw string) *url.URL {
		u, _ := url.Parse(raw)
		return u
	}

	t.Run("parses og tags", func(t *testing.T) {
		data, err := h.fetchOg(ts.URL, mustParse(ts.URL))
		if err != nil {
			t.Fatalf("fetchOg failed: %v", err)
		}
		if data.Title != "Test Page Title" {
			t.Errorf("title = %q, want %q", data.Title, "Test Page Title")
		}
		if data.Description != "A test description for the page" {
			t.Errorf("description = %q, want %q", data.Description, "A test description for the page")
		}
		if data.Image != "https://example.com/image.png" {
			t.Errorf("image = %q, want %q", data.Image, "https://example.com/image.png")
		}
		if data.SiteName != "Example Site" {
			t.Errorf("site_name = %q, want %q", data.SiteName, "Example Site")
		}
	})

	t.Run("falls back to title tag", func(t *testing.T) {
		ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html><head><title>Simple Title</title></head><body></body></html>`))
		}))
		defer ts2.Close()

		data, err := h.fetchOg(ts2.URL, mustParse(ts2.URL))
		if err != nil {
			t.Fatalf("fetchOg failed: %v", err)
		}
		if data.Title != "Simple Title" {
			t.Errorf("title = %q, want %q", data.Title, "Simple Title")
		}
	})

	t.Run("falls back to meta name description", func(t *testing.T) {
		ts3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html><head><meta name="description" content="Meta desc" /></head><body></body></html>`))
		}))
		defer ts3.Close()

		data, err := h.fetchOg(ts3.URL, mustParse(ts3.URL))
		if err != nil {
			t.Fatalf("fetchOg failed: %v", err)
		}
		if data.Description != "Meta desc" {
			t.Errorf("description = %q, want %q", data.Description, "Meta desc")
		}
	})

	t.Run("truncates long descriptions", func(t *testing.T) {
		long := strings.Repeat("x", 300)
		ts4 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html><head><meta property="og:description" content="` + long + `" /></head><body></body></html>`))
		}))
		defer ts4.Close()

		data, err := h.fetchOg(ts4.URL, mustParse(ts4.URL))
		if err != nil {
			t.Fatalf("fetchOg failed: %v", err)
		}
		if len(data.Description) > 200 {
			t.Errorf("description length = %d, want <= 200", len(data.Description))
		}
		if !strings.HasSuffix(data.Description, "...") {
			t.Errorf("description should end with ..., got %q", data.Description[len(data.Description)-10:])
		}
	})

	t.Run("returns favicon from google", func(t *testing.T) {
		data, err := h.fetchOg(ts.URL, mustParse(ts.URL))
		if err != nil {
			t.Fatalf("fetchOg failed: %v", err)
		}
		if !strings.Contains(data.Favicon, "google.com/s2/favicons") {
			t.Errorf("favicon = %q, want google favicons URL", data.Favicon)
		}
	})

	t.Run("handles empty page", func(t *testing.T) {
		ts5 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html><head></head><body></body></html>`))
		}))
		defer ts5.Close()

		data, err := h.fetchOg(ts5.URL, mustParse(ts5.URL))
		if err != nil {
			t.Fatalf("fetchOg failed: %v", err)
		}
		if data.Title != "" {
			t.Errorf("title = %q, want empty", data.Title)
		}
	})
}
