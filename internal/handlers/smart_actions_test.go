package handlers

import (
	"encoding/json"
	"testing"
)

func newTestSmartActionHandler() *SmartActionHandler {
	return &SmartActionHandler{}
}

func TestMapAction_RunTool(t *testing.T) {
	h := newTestSmartActionHandler()
	params, _ := json.Marshal(map[string]any{
		"tool": "list_features",
		"args": map[string]any{"project_id": "my-project"},
	})
	req := smartActionRequest{Type: "run_tool", Params: params}
	tool, args, err := h.mapAction(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "list_features" {
		t.Errorf("tool = %q, want list_features", tool)
	}
	if args["project_id"] != "my-project" {
		t.Errorf("args[project_id] = %v, want my-project", args["project_id"])
	}
}

func TestMapAction_RunTool_MissingName(t *testing.T) {
	h := newTestSmartActionHandler()
	params, _ := json.Marshal(map[string]any{"args": map[string]any{}})
	req := smartActionRequest{Type: "run_tool", Params: params}
	_, _, err := h.mapAction(req)
	if err == nil {
		t.Fatal("expected error when tool name is missing")
	}
}

func TestMapAction_RunBridge(t *testing.T) {
	h := newTestSmartActionHandler()
	params, _ := json.Marshal(map[string]any{
		"command": "git status",
		"args":    "--short",
	})
	req := smartActionRequest{Type: "run_bridge", Params: params}
	tool, args, err := h.mapAction(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "ai_prompt" {
		t.Errorf("tool = %q, want ai_prompt", tool)
	}
	prompt, _ := args["prompt"].(string)
	if prompt == "" {
		t.Error("expected non-empty prompt for run_bridge")
	}
}

func TestMapAction_RunBridge_MissingCommand(t *testing.T) {
	h := newTestSmartActionHandler()
	params, _ := json.Marshal(map[string]any{})
	req := smartActionRequest{Type: "run_bridge", Params: params}
	_, _, err := h.mapAction(req)
	if err == nil {
		t.Fatal("expected error when command is missing")
	}
}

func TestMapAction_FileRead(t *testing.T) {
	h := newTestSmartActionHandler()
	params, _ := json.Marshal(map[string]any{"path": "/workspace/README.md"})
	req := smartActionRequest{Type: "file_read", Params: params}
	tool, args, err := h.mapAction(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "read_file" {
		t.Errorf("tool = %q, want read_file", tool)
	}
	if args["path"] != "/workspace/README.md" {
		t.Errorf("args[path] = %v, want /workspace/README.md", args["path"])
	}
}

func TestMapAction_FileRead_MissingPath(t *testing.T) {
	h := newTestSmartActionHandler()
	params, _ := json.Marshal(map[string]any{})
	req := smartActionRequest{Type: "file_read", Params: params}
	_, _, err := h.mapAction(req)
	if err == nil {
		t.Fatal("expected error when path is missing")
	}
}

func TestMapAction_FileWrite(t *testing.T) {
	h := newTestSmartActionHandler()
	params, _ := json.Marshal(map[string]any{
		"path":    "/workspace/out.txt",
		"content": "hello world",
	})
	req := smartActionRequest{Type: "file_write", Params: params}
	tool, args, err := h.mapAction(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "write_file" {
		t.Errorf("tool = %q, want write_file", tool)
	}
	if args["content"] != "hello world" {
		t.Errorf("args[content] = %v, want hello world", args["content"])
	}
}

func TestMapAction_FileWrite_MissingPath(t *testing.T) {
	h := newTestSmartActionHandler()
	params, _ := json.Marshal(map[string]any{"content": "data"})
	req := smartActionRequest{Type: "file_write", Params: params}
	_, _, err := h.mapAction(req)
	if err == nil {
		t.Fatal("expected error when path is missing")
	}
}

func TestMapAction_RunTool_NilArgs(t *testing.T) {
	h := newTestSmartActionHandler()
	params, _ := json.Marshal(map[string]any{"tool": "sync_status"})
	req := smartActionRequest{Type: "run_tool", Params: params}
	tool, args, err := h.mapAction(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "sync_status" {
		t.Errorf("tool = %q, want sync_status", tool)
	}
	if args == nil {
		t.Error("args should not be nil when not provided (defaults to empty map)")
	}
}

func TestMapAction_Unsupported(t *testing.T) {
	h := newTestSmartActionHandler()
	req := smartActionRequest{Type: "unknown_action"}
	_, _, err := h.mapAction(req)
	if err == nil {
		t.Fatal("expected error for unsupported action type")
	}
}

func TestMapAction_ExistingTypes(t *testing.T) {
	h := newTestSmartActionHandler()

	cases := []struct {
		actionType string
		params     map[string]any
		wantTool   string
	}{
		{"sync_workspace", nil, "sync_now"},
		{"get_status", nil, "sync_status"},
		{"list_tools", nil, "sync_status"},
	}

	for _, tc := range cases {
		var params json.RawMessage
		if tc.params != nil {
			params, _ = json.Marshal(tc.params)
		}
		req := smartActionRequest{Type: tc.actionType, Params: params}
		tool, _, err := h.mapAction(req)
		if err != nil {
			t.Errorf("[%s] unexpected error: %v", tc.actionType, err)
			continue
		}
		if tool != tc.wantTool {
			t.Errorf("[%s] tool = %q, want %q", tc.actionType, tool, tc.wantTool)
		}
	}
}
