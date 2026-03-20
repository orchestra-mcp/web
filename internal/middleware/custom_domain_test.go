package middleware

import "testing"

func TestIsStandardHost(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"localhost", true},
		{"orchestra.dev", true},
		{"orchestra.local", true},
		{"127.0.0.1", true},
		{"app.orchestra.dev", true},
		{"sub.orchestra.local", true},
		{"api.orchestra.dev", true},
		{"example.com", false},
		{"myapp.com", false},
		{"orchestradev", false},        // not a subdomain match
		{"notorchestra.dev", false},     // different prefix, no dot separator
		{"orchestra.dev.evil.com", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isStandardHost(tt.host)
		if got != tt.expected {
			t.Errorf("isStandardHost(%q) = %v, want %v", tt.host, got, tt.expected)
		}
	}
}
