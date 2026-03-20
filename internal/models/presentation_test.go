package models

import "testing"

func TestPresentationDefaults(t *testing.T) {
	p := Presentation{}

	if p.Visibility != "" {
		t.Errorf("expected empty default Visibility, got %q", p.Visibility)
	}
	if p.SlideCount != 0 {
		t.Errorf("expected zero default SlideCount, got %d", p.SlideCount)
	}
	if p.Version != 0 {
		t.Errorf("expected zero default Version, got %d", p.Version)
	}
}

func TestPresentationSlideDefaults(t *testing.T) {
	s := PresentationSlide{}

	if s.Layout != "" {
		t.Errorf("expected empty default Layout, got %q", s.Layout)
	}
	if s.SlideNumber != 0 {
		t.Errorf("expected zero default SlideNumber, got %d", s.SlideNumber)
	}
	if s.PresentationID != "" {
		t.Errorf("expected empty default PresentationID, got %q", s.PresentationID)
	}
}

func TestPresentationSlideLayouts(t *testing.T) {
	validLayouts := []string{"title", "title-content", "two-column", "image-full", "quote", "blank"}
	for _, layout := range validLayouts {
		s := PresentationSlide{Layout: layout}
		if s.Layout != layout {
			t.Errorf("expected layout %q, got %q", layout, s.Layout)
		}
	}
}
