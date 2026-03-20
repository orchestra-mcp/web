package models

import "testing"

func TestSharedContentNewFields(t *testing.T) {
	sc := SharedContent{}

	if sc.EntityID != "" {
		t.Errorf("expected empty default EntityID, got %q", sc.EntityID)
	}
	if sc.UniqueViews != 0 {
		t.Errorf("expected zero default UniqueViews, got %d", sc.UniqueViews)
	}
	if sc.CustomDomain != "" {
		t.Errorf("expected empty default CustomDomain, got %q", sc.CustomDomain)
	}
}

func TestSharedContentEntityTypes(t *testing.T) {
	types := []string{"note", "skill", "agent", "workflow", "prompt", "api_collection", "presentation", "doc"}
	for _, et := range types {
		sc := SharedContent{EntityType: et}
		if sc.EntityType != et {
			t.Errorf("expected entity_type %q, got %q", et, sc.EntityType)
		}
	}
}
