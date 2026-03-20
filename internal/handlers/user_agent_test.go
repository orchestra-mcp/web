package handlers

import "testing"

func TestParseUserAgent(t *testing.T) {
	tests := []struct {
		ua              string
		wantDevice, wantOS, wantBrowser string
	}{
		{
			ua:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantDevice:  "Mac",
			wantOS:      "macOS",
			wantBrowser: "Chrome",
		},
		{
			ua:          "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			wantDevice:  "iPhone",
			wantOS:      "iOS",
			wantBrowser: "Safari",
		},
		{
			ua:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			wantDevice:  "PC",
			wantOS:      "Windows",
			wantBrowser: "Edge",
		},
		{
			ua:          "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0",
			wantDevice:  "Linux PC",
			wantOS:      "Linux",
			wantBrowser: "Firefox",
		},
		{
			ua:          "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			wantDevice:  "Android Device",
			wantOS:      "Android",
			wantBrowser: "Chrome",
		},
		{
			ua:          "Orchestra/1.0 (macOS)",
			wantDevice:  "Mac",
			wantOS:      "macOS",
			wantBrowser: "Orchestra App",
		},
	}

	for _, tt := range tests {
		device, os, browser := parseUserAgent(tt.ua)
		if device != tt.wantDevice {
			t.Errorf("parseUserAgent(%q): device = %q, want %q", tt.ua, device, tt.wantDevice)
		}
		if os != tt.wantOS {
			t.Errorf("parseUserAgent(%q): os = %q, want %q", tt.ua, os, tt.wantOS)
		}
		if browser != tt.wantBrowser {
			t.Errorf("parseUserAgent(%q): browser = %q, want %q", tt.ua, browser, tt.wantBrowser)
		}
	}
}

func TestClassifyDevice(t *testing.T) {
	tests := []struct {
		ua   string
		want string
	}{
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)", "mobile"},
		{"Mozilla/5.0 (iPad; CPU OS 17_0)", "tablet"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X)", "desktop"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64)", "desktop"},
		{"Mozilla/5.0 (Linux; Android 14; Pixel 8) Mobile", "mobile"},
	}

	for _, tt := range tests {
		got := classifyDevice(tt.ua)
		if got != tt.want {
			t.Errorf("classifyDevice(%q): got %q, want %q", tt.ua, got, tt.want)
		}
	}
}
