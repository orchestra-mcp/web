package handlers

import "testing"

func TestNewApiCollectionHandler(t *testing.T) {
	h := NewApiCollectionHandler(nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.db != nil {
		t.Error("expected nil db when constructed with nil")
	}
}

func TestApiCollectionAllowedTablesIncludesNewTables(t *testing.T) {
	tables := []string{"api_collections", "api_endpoints", "api_environments"}
	for _, table := range tables {
		if !allowedTables[table] {
			t.Errorf("allowedTables missing %q — PowerSync CRUD won't work for this table", table)
		}
	}
}
