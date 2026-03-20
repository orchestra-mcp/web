package handlers

import (
	"encoding/json"
	"testing"
)

// TestExtractMCPOutput_TextContent verifies text is extracted from MCP content array.
func TestExtractMCPOutput_TextContent(t *testing.T) {
	result := json.RawMessage(`{"content":[{"type":"text","text":"hello world"}],"isError":false}`)
	got := extractMCPOutput(result)
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

// TestExtractMCPOutput_MultipleItems concatenates multiple text items with newline.
func TestExtractMCPOutput_MultipleItems(t *testing.T) {
	result := json.RawMessage(`{"content":[{"type":"text","text":"line1"},{"type":"text","text":"line2"}]}`)
	got := extractMCPOutput(result)
	if got != "line1\nline2" {
		t.Errorf("got %q, want %q", got, "line1\nline2")
	}
}

// TestExtractMCPOutput_SkipsNonText ignores non-text content items.
func TestExtractMCPOutput_SkipsNonText(t *testing.T) {
	result := json.RawMessage(`{"content":[{"type":"image","url":"x"},{"type":"text","text":"ok"}]}`)
	got := extractMCPOutput(result)
	if got != "ok" {
		t.Errorf("got %q, want %q", got, "ok")
	}
}

// TestExtractMCPOutput_NilResult returns empty string for nil input.
func TestExtractMCPOutput_NilResult(t *testing.T) {
	got := extractMCPOutput(nil)
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

// TestExtractMCPOutput_InvalidJSON falls back to raw string.
func TestExtractMCPOutput_InvalidJSON(t *testing.T) {
	result := json.RawMessage(`not-json`)
	got := extractMCPOutput(result)
	if got != "not-json" {
		t.Errorf("got %q, want raw fallback", got)
	}
}

// TestExtractMCPFiles_ParsesCreatedLines extracts "Created: " prefixed lines.
func TestExtractMCPFiles_ParsesCreatedLines(t *testing.T) {
	result := json.RawMessage(`{"content":[{"type":"text","text":"Created: /foo/bar.go\nCreated: /baz/qux.go"}]}`)
	files := extractMCPFiles(result)
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2: %v", len(files), files)
	}
	if files[0] != "/foo/bar.go" {
		t.Errorf("files[0] = %q, want /foo/bar.go", files[0])
	}
	if files[1] != "/baz/qux.go" {
		t.Errorf("files[1] = %q, want /baz/qux.go", files[1])
	}
}

// TestExtractMCPFiles_ParsesModifiedLines extracts "Modified: " prefixed lines.
func TestExtractMCPFiles_ParsesModifiedLines(t *testing.T) {
	result := json.RawMessage(`{"content":[{"type":"text","text":"Modified: /src/main.go"}]}`)
	files := extractMCPFiles(result)
	if len(files) != 1 || files[0] != "/src/main.go" {
		t.Errorf("got %v, want [\"/src/main.go\"]", files)
	}
}

// TestExtractMCPFiles_EmptyWhenNoMatches returns nil when no file lines found.
func TestExtractMCPFiles_EmptyWhenNoMatches(t *testing.T) {
	result := json.RawMessage(`{"content":[{"type":"text","text":"done, no files"}]}`)
	files := extractMCPFiles(result)
	if len(files) != 0 {
		t.Errorf("got %v, want empty", files)
	}
}

// TestSplitLines_Basic splits on newlines correctly.
func TestSplitLines_Basic(t *testing.T) {
	lines := splitLines("a\nb\nc")
	if len(lines) != 3 || lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
		t.Errorf("got %v", lines)
	}
}

// TestSplitLines_Empty returns nil for empty string.
func TestSplitLines_Empty(t *testing.T) {
	if splitLines("") != nil {
		t.Error("expected nil for empty string")
	}
}

// TestBridgeLongRunningTools verifies the long-running tool map contents.
func TestBridgeLongRunningTools(t *testing.T) {
	for _, tool := range []string{"run_prompt", "run_command", "run_bridge", "run_tests"} {
		if !bridgeLongRunningTools[tool] {
			t.Errorf("expected %q in bridgeLongRunningTools", tool)
		}
	}
	if bridgeLongRunningTools["get_status"] {
		t.Error("get_status should not be in bridgeLongRunningTools")
	}
}

// TestTunnelHub_ProgressChannel verifies ForwardToBrowser routes progress envelopes
// to progressCh and final responses to respCh.
func TestTunnelHub_ProgressChannel(t *testing.T) {
	hub := NewTunnelHub()
	respCh := make(chan json.RawMessage, 1)
	progCh := make(chan string, 4)

	hub.RegisterSmartAction("sess1", respCh, progCh)
	defer hub.UnregisterSmartAction("sess1")

	// Send a progress envelope.
	progressMsg := json.RawMessage(`{"type":"progress","text":"working…"}`)
	if err := hub.ForwardToBrowser("sess1", progressMsg); err != nil {
		t.Fatalf("ForwardToBrowser progress: %v", err)
	}
	select {
	case txt := <-progCh:
		if txt != "working…" {
			t.Errorf("progressCh got %q, want working…", txt)
		}
	default:
		t.Error("expected progress message in progCh")
	}
	if len(respCh) != 0 {
		t.Error("respCh should be empty after progress message")
	}

	// Send a final response.
	finalMsg := json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":{}}`)
	if err := hub.ForwardToBrowser("sess1", finalMsg); err != nil {
		t.Fatalf("ForwardToBrowser final: %v", err)
	}
	select {
	case <-respCh:
		// good
	default:
		t.Error("expected final message in respCh")
	}
}

// TestTunnelHub_NilProgressChannel verifies that nil progressCh is safe (all messages go to respCh).
func TestTunnelHub_NilProgressChannel(t *testing.T) {
	hub := NewTunnelHub()
	respCh := make(chan json.RawMessage, 2)
	hub.RegisterSmartAction("sess2", respCh, nil)
	defer hub.UnregisterSmartAction("sess2")

	// Even a "progress"-typed message goes to respCh when no progressCh.
	progressMsg := json.RawMessage(`{"type":"progress","text":"x"}`)
	_ = hub.ForwardToBrowser("sess2", progressMsg)
	if len(respCh) != 1 {
		t.Error("expected message in respCh when progressCh is nil")
	}
}

// TestTunnelHub_UnknownSession returns nil (not an error) for unknown sessions.
func TestTunnelHub_UnknownSession(t *testing.T) {
	hub := NewTunnelHub()
	err := hub.ForwardToBrowser("no-such-session", json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("expected nil error for unknown session, got %v", err)
	}
}
