package handlers

import (
	"testing"
)

func TestAllowedTablesContainsCorrectHealthTables(t *testing.T) {
	correctNames := []string{
		"water_logs",
		"caffeine_logs",
		"meal_logs",
		"pomodoro_sessions",
		"sleep_configs",
		"health_snapshots",
		"health_profiles",
		"sleep_logs",
	}

	for _, name := range correctNames {
		if !allowedTables[name] {
			t.Errorf("allowedTables missing correct table %q", name)
		}
	}
}

func TestAllowedTablesRejectsLegacyHealthTables(t *testing.T) {
	legacyNames := []string{
		"health_hydration",
		"health_caffeine",
		"health_nutrition",
		"health_pomodoro",
		"health_shutdown",
		"health_weight",
		"health_sleep",
		"health_vitals",
		"health_settings",
	}

	for _, name := range legacyNames {
		if allowedTables[name] {
			t.Errorf("allowedTables should NOT contain legacy table %q", name)
		}
	}
}

func TestAllowedTablesContainsAppDataTables(t *testing.T) {
	appTables := []string{
		"notes",
		"projects",
		"features",
		"plans",
		"requests",
		"persons",
		"agents",
		"skills",
		"workflows",
		"docs",
		"delegations",
		"sessions",
		"user_settings",
		"workspaces",
		"teams",
		"memberships",
	}

	for _, name := range appTables {
		if !allowedTables[name] {
			t.Errorf("allowedTables missing app table %q", name)
		}
	}
}

func TestQuoteEscapesDoubleQuotes(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"water_logs", "\"water_logs\""},
		{"my\"table", "\"my\"\"table\""},
	}

	for _, tc := range cases {
		got := quote(tc.input)
		if got != tc.expected {
			t.Errorf("quote(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
