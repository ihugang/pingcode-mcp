// These offline HTTP/MCP tests cover bootstrap-pingcode-mcp task 4.2 and PRD
// FR-003, FR-006, and FR-007. PingCode demand ID: PENDING.
package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ihugang/pingcode-mcp-go/internal/pingcode"
	"github.com/ihugang/pingcode-mcp-go/internal/security"
)

type fixedTokenProvider struct{}

func (fixedTokenProvider) AccessToken(context.Context) (string, error) { return "test-token", nil }

func TestMCPWithPingCodeClientListWorkItemsOverHTTPMock(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/pjm/workitems" {
			t.Errorf("request = %s %s, want GET /v1/pjm/workitems", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("project_id") != "p1" || r.URL.Query().Get("page_size") != "30" {
			t.Errorf("query = %v", r.URL.Query())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"page_size":30,"page_index":0,"total":1,"values":[{"id":"w1","title":"mock item","project":{"id":"p1"},"future_field":"kept"}]}`))
	}))
	defer upstream.Close()

	client := newPingCodeHarness(t, upstream.URL)
	result, err := client.CallTool(context.Background(), &sdk.CallToolParams{
		Name: "pingcode_list_work_items", Arguments: map[string]any{"project_id": "p1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", textContent(t, result))
	}
	output := resultObject(t, result.StructuredContent)
	if output["page_size"] != float64(30) || output["page_index"] != float64(0) || output["total"] != float64(1) {
		t.Fatalf("pagination output = %#v", output)
	}
	values := output["values"].([]any)
	if len(values) != 1 || values[0].(map[string]any)["future_field"] != "kept" {
		t.Fatalf("work item Raw was not preserved: %#v", values)
	}
}

func TestMCPWithPingCodeClientRejectsWorkItemProjectMismatch(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/pjm/workitems/w1" {
			t.Errorf("request = %s %s, want GET /v1/pjm/workitems/w1", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"w1","title":"must not leak","project":{"id":"p2"},"future_field":"private"}`))
	}))
	defer upstream.Close()

	client := newPingCodeHarness(t, upstream.URL)
	result, err := client.CallTool(context.Background(), &sdk.CallToolParams{
		Name: "pingcode_get_work_item", Arguments: map[string]any{"workitem_id": "w1", "project_id": "p1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := errorType(t, result); got != ErrorProjectNotAllowed {
		t.Fatalf("error type = %q, want %q", got, ErrorProjectNotAllowed)
	}
	if text := textContent(t, result); containsAny(text, "must not leak", "private") {
		t.Fatalf("mismatched work item data leaked: %s", text)
	}
}

func newPingCodeHarness(t *testing.T, baseURL string) *sdk.ClientSession {
	t.Helper()
	policy := security.NewPolicy(false, []string{"p1"})
	api := pingcode.New(fixedTokenProvider{}, policy, &http.Client{Timeout: 5 * time.Second}, baseURL)
	return newHarness(t, api, policy)
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
