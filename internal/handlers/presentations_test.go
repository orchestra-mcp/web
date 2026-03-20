package handlers

import "testing"

func TestNewPresentationHandler(t *testing.T) {
	h := NewPresentationHandler(nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.db != nil {
		t.Error("expected nil db when constructed with nil")
	}
	if h.exports == nil {
		t.Error("expected non-nil exports service")
	}
}

func TestPresentationAllowedTablesIncludesNewTables(t *testing.T) {
	tables := []string{"presentations", "presentation_slides"}
	for _, table := range tables {
		if !allowedTables[table] {
			t.Errorf("allowedTables missing %q — PowerSync CRUD won't work for this table", table)
		}
	}
}
