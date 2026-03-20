package models

import "testing"

func TestApiCollectionDefaults(t *testing.T) {
	col := ApiCollection{}

	if col.AuthType != "" {
		t.Errorf("expected empty default AuthType, got %q", col.AuthType)
	}
	if col.Visibility != "" {
		t.Errorf("expected empty default Visibility, got %q", col.Visibility)
	}
	if col.Version != 0 {
		t.Errorf("expected zero default Version, got %d", col.Version)
	}
}

func TestApiEndpointDefaults(t *testing.T) {
	ep := ApiEndpoint{}

	if ep.Method != "" {
		t.Errorf("expected empty default Method, got %q", ep.Method)
	}
	if ep.SortOrder != 0 {
		t.Errorf("expected zero default SortOrder, got %d", ep.SortOrder)
	}
}

func TestApiEnvironmentDefaults(t *testing.T) {
	env := ApiEnvironment{}

	if env.IsActive {
		t.Error("expected IsActive to default to false")
	}
	if env.Name != "" {
		t.Errorf("expected empty default Name, got %q", env.Name)
	}
}
