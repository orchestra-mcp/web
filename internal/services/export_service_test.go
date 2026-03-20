package services

import (
	"strings"
	"testing"

	"github.com/orchestra-mcp/web/internal/models"
)

func TestRenderPresentationHTML(t *testing.T) {
	svc := NewExportService()

	pres := models.Presentation{Title: "Test Deck"}
	slides := []models.PresentationSlide{
		{SlideNumber: 1, Layout: "title", Title: "Welcome", Content: "Hello world"},
		{SlideNumber: 2, Layout: "title-content", Title: "Agenda", Content: "- Item 1\n- Item 2"},
	}

	html := svc.RenderPresentationHTML(pres, slides)

	if !strings.Contains(html, "Test Deck") {
		t.Error("HTML should contain presentation title")
	}
	if !strings.Contains(html, "Welcome") {
		t.Error("HTML should contain slide 1 title")
	}
	if !strings.Contains(html, "Agenda") {
		t.Error("HTML should contain slide 2 title")
	}
	if !strings.Contains(html, "<li>") {
		t.Error("HTML should render markdown list items as <li>")
	}
	if !strings.Contains(html, "1 / 2") {
		t.Error("HTML should contain slide numbering")
	}
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML should be a full document")
	}
}

func TestGeneratePPTXMarkdown(t *testing.T) {
	svc := NewExportService()

	pres := models.Presentation{Title: "My Presentation", Description: "A test deck"}
	slides := []models.PresentationSlide{
		{SlideNumber: 1, Title: "Intro", Content: "Welcome all", Notes: "Say hello"},
		{SlideNumber: 2, Title: "Main", Content: "Key points"},
	}

	md := svc.GeneratePPTXMarkdown(pres, slides)

	if !strings.HasPrefix(md, "# My Presentation") {
		t.Error("PPTX markdown should start with title")
	}
	if !strings.Contains(md, "A test deck") {
		t.Error("PPTX markdown should contain description")
	}
	if !strings.Contains(md, "## Slide 1: Intro") {
		t.Error("PPTX markdown should contain slide 1 heading")
	}
	if !strings.Contains(md, "**Notes:** Say hello") {
		t.Error("PPTX markdown should contain presenter notes")
	}
	if !strings.Contains(md, "---") {
		t.Error("PPTX markdown should contain slide separators")
	}
}

func TestGoogleSlidesURL(t *testing.T) {
	svc := NewExportService()

	pres := models.Presentation{Title: "My Cool Deck"}
	url := svc.GoogleSlidesURL(pres)

	if !strings.HasPrefix(url, "https://docs.google.com/presentation/create") {
		t.Errorf("expected Google Slides URL, got %q", url)
	}
	if !strings.Contains(url, "My+Cool+Deck") {
		t.Error("URL should contain URL-encoded title")
	}
}

func TestSlideHTML(t *testing.T) {
	svc := NewExportService()

	slide := models.PresentationSlide{
		Layout:  "quote",
		Title:   "Famous Words",
		Content: "To be or not to be",
	}

	html := svc.SlideHTML(slide)

	if !strings.Contains(html, `data-layout="quote"`) {
		t.Error("slide HTML should include layout data attribute")
	}
	if !strings.Contains(html, "Famous Words") {
		t.Error("slide HTML should include title")
	}
	if !strings.Contains(html, "To be or not to be") {
		t.Error("slide HTML should include content")
	}
}
