package handlers

import "testing"

func TestMapEventAction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"tool_use_start", "tool_called"},
		{"tool_use_end", "tool_called"},
		{"agent_tool_use_start", "tool_called"},
		{"agent_tool_use_end", "tool_called"},
		{"notification", "notification"},
		{"subagent_start", "agent_spawned"},
		{"subagent_end", "agent_spawned"},
		{"custom_event", "custom_event"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapEventAction(tt.input)
			if result != tt.expected {
				t.Errorf("mapEventAction(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEntityTypeFromEvent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"tool_use_start", "tool"},
		{"tool_use_end", "tool"},
		{"notification", "notification"},
		{"subagent_start", "agent"},
		{"subagent_end", "agent"},
		{"unknown", "mcp_event"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := entityTypeFromEvent(tt.input)
			if result != tt.expected {
				t.Errorf("entityTypeFromEvent(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
