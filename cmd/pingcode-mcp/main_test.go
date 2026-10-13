// Tests cover the stdio protocol contract in bootstrap-pingcode-mcp task 4.2
// and PRD FR-003. PingCode demand ID: PENDING.
package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestStdioCommandTransportKeepsStdoutProtocolClean(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "pingcode-mcp")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary)
	command.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		"PINGCODE_ALLOWED_PROJECT_IDS=project-1",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := sdk.NewClient(&sdk.Implementation{Name: "stdio-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, &sdk.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatalf("stdio initialize failed (stdout must contain only MCP protocol): %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	tools := 0
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			t.Fatal(err)
		}
		tools++
		if tool.Description == "" || tool.InputSchema == nil {
			t.Errorf("tool %q lacks description or input schema", tool.Name)
		}
	}
	if tools != 6 {
		t.Fatalf("tools/list returned %d tools, want 6", tools)
	}

	call := func(name string, args map[string]any) *sdk.CallToolResult {
		t.Helper()
		result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("CallTool(%s): %v", name, err)
		}
		return result
	}
	for _, tc := range []struct {
		name string
		args map[string]any
		kind string
	}{
		{name: "pingcode_get_project", args: map[string]any{"project_id": "project-1"}, kind: "AUTHENTICATION"},
		{name: "pingcode_get_project", args: map[string]any{"project_id": "other"}, kind: "PROJECT_NOT_ALLOWED"},
		{name: "pingcode_create_work_item", args: map[string]any{"project_id": "project-1", "type_id": "task", "title": "t"}, kind: "READ_ONLY"},
	} {
		result := call(tc.name, tc.args)
		if !result.IsError {
			t.Fatalf("%s unexpectedly succeeded", tc.name)
		}
		var structured map[string]any
		encoded, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(encoded, &structured); err != nil {
			t.Fatalf("structured error is not JSON: %v", err)
		}
		if got := structured["error"].(map[string]any)["type"]; got != tc.kind {
			t.Errorf("%s error type = %#v, want %q", tc.name, got, tc.kind)
		}
	}
}
