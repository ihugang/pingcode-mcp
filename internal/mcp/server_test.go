package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ihugang/pingcode-mcp-go/internal/auth"
	"github.com/ihugang/pingcode-mcp-go/internal/pingcode"
	"github.com/ihugang/pingcode-mcp-go/internal/security"
)

type fakeAPI struct {
	mu         sync.Mutex
	err        error
	calls      map[string]int
	projects   pingcode.Page[pingcode.Project]
	project    pingcode.Project
	workPage   pingcode.Page[pingcode.WorkItem]
	workItem   pingcode.WorkItem
	workParams pingcode.ListWorkItemsParams
}

func newFakeAPI() *fakeAPI {
	return &fakeAPI{
		calls: map[string]int{},
		projects: pingcode.Page[pingcode.Project]{PageSize: 3, PageIndex: 1, Total: 3, Values: []pingcode.Project{
			{ID: "p1", Name: "one", Raw: json.RawMessage(`{"id":"p1","name":"one","future_project":"kept"}`)},
			{ID: "p2", Name: "two", Raw: json.RawMessage(`{"id":"p2","name":"two"}`)},
			{ID: "p3", Name: "three", Raw: json.RawMessage(`{"id":"p3","name":"three"}`)},
		}},
		project: pingcode.Project{ID: "p1", Name: "one", Raw: json.RawMessage(`{"id":"p1","name":"one","future_project":"kept"}`)},
		workPage: pingcode.Page[pingcode.WorkItem]{PageSize: 30, PageIndex: 0, Total: 1, Values: []pingcode.WorkItem{
			{ID: "w1", Title: "item", Raw: json.RawMessage(`{"id":"w1","title":"item","future_work_item":"kept"}`)},
		}},
		workItem: pingcode.WorkItem{ID: "w1", Title: "item", Raw: json.RawMessage(`{"id":"w1","title":"item","future_work_item":"kept"}`)},
	}
}

func (f *fakeAPI) result(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[name]++
	return f.err
}

func (f *fakeAPI) callCount(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[name]
}

func (f *fakeAPI) ListProjects(context.Context, pingcode.ListProjectsParams) (pingcode.Page[pingcode.Project], error) {
	if err := f.result("list_projects"); err != nil {
		return pingcode.Page[pingcode.Project]{}, err
	}
	return f.projects, nil
}

func (f *fakeAPI) GetProject(context.Context, string, pingcode.GetOptions) (pingcode.Project, error) {
	if err := f.result("get_project"); err != nil {
		return pingcode.Project{}, err
	}
	return f.project, nil
}

func (f *fakeAPI) ListWorkItems(_ context.Context, params pingcode.ListWorkItemsParams) (pingcode.Page[pingcode.WorkItem], error) {
	f.mu.Lock()
	f.workParams = params
	f.mu.Unlock()
	if err := f.result("list_work_items"); err != nil {
		return pingcode.Page[pingcode.WorkItem]{}, err
	}
	return f.workPage, nil
}

func (f *fakeAPI) GetWorkItem(context.Context, string, string, pingcode.GetOptions) (pingcode.WorkItem, error) {
	if err := f.result("get_work_item"); err != nil {
		return pingcode.WorkItem{}, err
	}
	return f.workItem, nil
}

func (f *fakeAPI) CreateWorkItem(context.Context, pingcode.CreateWorkItemRequest) (pingcode.WorkItem, error) {
	if err := f.result("create_work_item"); err != nil {
		return pingcode.WorkItem{}, err
	}
	return f.workItem, nil
}

func (f *fakeAPI) UpdateWorkItem(context.Context, string, string, pingcode.UpdateWorkItemRequest) (pingcode.WorkItem, error) {
	if err := f.result("update_work_item"); err != nil {
		return pingcode.WorkItem{}, err
	}
	return f.workItem, nil
}

func newHarness(t *testing.T, api PingCodeAPI, policy *security.Policy) *sdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	server := NewServer(api, policy, Options{Version: "test"})
	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "test"}, nil)
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	})
	return clientSession
}

func callTool(t *testing.T, client *sdk.ClientSession, name string, args map[string]any) *sdk.CallToolResult {
	t.Helper()
	result, err := client.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s) error: %v", name, err)
	}
	return result
}

func TestServerRegistersSixToolsAndContractSchemas(t *testing.T) {
	client := newHarness(t, newFakeAPI(), security.NewPolicy(false, []string{"p1"}))
	var tools []*sdk.Tool
	for tool, err := range client.Tools(context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		tools = append(tools, tool)
	}
	wantNames := []string{
		"pingcode_list_projects", "pingcode_get_project", "pingcode_list_work_items",
		"pingcode_get_work_item", "pingcode_create_work_item", "pingcode_update_work_item",
	}
	if len(tools) != len(wantNames) {
		t.Fatalf("tool count = %d, want %d", len(tools), len(wantNames))
	}
	byName := make(map[string]*sdk.Tool, len(tools))
	for _, tool := range tools {
		if tool.Description == "" || tool.InputSchema == nil {
			t.Errorf("tool %q lacks description or input schema", tool.Name)
		}
		if strings.Contains(strings.ToLower(tool.Name), "delete") || strings.Contains(strings.ToLower(tool.Name), "remove") {
			t.Errorf("unexpected destructive tool %q", tool.Name)
		}
		byName[tool.Name] = tool
	}
	for _, name := range wantNames {
		if byName[name] == nil {
			t.Errorf("missing tool %q", name)
		}
		if got := schemaMap(t, byName[name].InputSchema)["additionalProperties"]; got != false {
			t.Errorf("%s additionalProperties = %#v, want false", name, got)
		}
	}

	create := schemaMap(t, byName["pingcode_create_work_item"].InputSchema)
	assertRequired(t, create, "project_id", "type_id", "title")
	createProps := properties(t, create)
	assertHasKeys(t, createProps, "sprint_id", "participant_ids")
	assertMissingKeys(t, createProps, "phase_id")

	getWork := schemaMap(t, byName["pingcode_get_work_item"].InputSchema)
	assertRequired(t, getWork, "workitem_id", "project_id")
	getProject := schemaMap(t, byName["pingcode_get_project"].InputSchema)
	assertRequired(t, getProject, "project_id")
	listProjects := schemaMap(t, byName["pingcode_list_projects"].InputSchema)
	if required, ok := listProjects["required"].([]any); ok && len(required) != 0 {
		t.Fatalf("list_projects has required fields: %#v", required)
	}
	update := schemaMap(t, byName["pingcode_update_work_item"].InputSchema)
	assertRequired(t, update, "workitem_id", "project_id")
	updateProps := properties(t, update)
	// The update schema must exclude create-only sprint and participant fields.
	assertHasKeys(t, updateProps, "phase_id")
	assertMissingKeys(t, updateProps, "sprint_id", "participant_ids")
	assertExactKeys(t, updateProps,
		"workitem_id", "project_id", "title", "description", "start_at", "end_at", "priority_id", "state_id",
		"assignee_id", "parent_id", "release_ids", "board_id", "entry_id", "swimlane_id", "phase_id",
		"story_points", "estimated_workload", "remaining_workload", "properties")

	listWork := schemaMap(t, byName["pingcode_list_work_items"].InputSchema)
	assertRequired(t, listWork, "project_id")
	for _, name := range []string{"pingcode_list_projects", "pingcode_list_work_items"} {
		props := properties(t, schemaMap(t, byName[name].InputSchema))
		pageSize := props["page_size"].(map[string]any)
		if pageSize["minimum"] != float64(0) || pageSize["maximum"] != float64(100) {
			t.Errorf("%s page_size bounds = %#v", name, pageSize)
		}
		pageIndex := props["page_index"].(map[string]any)
		if pageIndex["minimum"] != float64(0) {
			t.Errorf("%s page_index minimum = %#v", name, pageIndex["minimum"])
		}
		root := schemaMap(t, byName[name].InputSchema)
		if root["additionalProperties"] != false {
			t.Errorf("%s additionalProperties = %#v, want false", name, root["additionalProperties"])
		}
	}
}

func TestSixToolsReturnResultsAndPreserveRawFields(t *testing.T) {
	client := newHarness(t, newFakeAPI(), security.NewPolicy(false, []string{"p1"}))
	cases := []struct {
		name string
		args map[string]any
		key  string
		id   string
	}{
		{name: "pingcode_list_projects", key: "values", id: "p1"},
		{name: "pingcode_get_project", args: map[string]any{"project_id": "p1"}, key: "project", id: "p1"},
		{name: "pingcode_list_work_items", args: map[string]any{"project_id": "p1"}, key: "values", id: "w1"},
		{name: "pingcode_get_work_item", args: map[string]any{"workitem_id": "w1", "project_id": "p1"}, key: "work_item", id: "w1"},
		{name: "pingcode_create_work_item", args: map[string]any{"project_id": "p1", "type_id": "task", "title": "new"}, key: "work_item", id: "w1"},
		{name: "pingcode_update_work_item", args: map[string]any{"workitem_id": "w1", "project_id": "p1", "title": "changed"}, key: "work_item", id: "w1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := callTool(t, client, tc.name, tc.args)
			if result.IsError {
				t.Fatalf("tool error: %s", textContent(t, result))
			}
			structured := resultObject(t, result.StructuredContent)
			content := map[string]any{}
			if err := json.Unmarshal([]byte(textContent(t, result)), &content); err != nil {
				t.Fatalf("content is not JSON: %v", err)
			}
			if !reflect.DeepEqual(content, structured) {
				t.Fatalf("content and structured output differ: content=%#v structured=%#v", content, structured)
			}
			if tc.key == "values" {
				for _, field := range []string{"page_size", "page_index", "total", "values"} {
					if _, ok := structured[field]; !ok {
						t.Errorf("%s output missing %s", tc.name, field)
					}
				}
				if tc.name == "pingcode_list_projects" && structured["page_size"] != float64(3) {
					t.Errorf("project page_size = %#v, want response value 3", structured["page_size"])
				}
				values := structured["values"].([]any)
				if len(values) == 0 || values[0].(map[string]any)["id"] != tc.id {
					t.Fatalf("unexpected values: %#v", values)
				}
				if tc.name == "pingcode_list_projects" && values[0].(map[string]any)["future_project"] != "kept" {
					t.Fatal("project Raw field was not passed through")
				}
				if tc.name == "pingcode_list_work_items" && values[0].(map[string]any)["future_work_item"] != "kept" {
					t.Fatal("work item Raw field was not passed through")
				}
			} else if resource := structured[tc.key].(map[string]any); resource["id"] != tc.id {
				t.Errorf("%s resource ID = %#v, want %q", tc.name, resource["id"], tc.id)
			}
		})
	}
}

func TestListProjectsFiltersAllowlistAndSkipsEmptyAllowlist(t *testing.T) {
	api := newFakeAPI()
	client := newHarness(t, api, security.NewPolicy(false, []string{"p1"}))
	result := callTool(t, client, "pingcode_list_projects", nil)
	if result.IsError {
		t.Fatal(textContent(t, result))
	}
	output := resultObject(t, result.StructuredContent)
	values := output["values"].([]any)
	if len(values) != 1 || values[0].(map[string]any)["id"] != "p1" || output["total"] != float64(1) {
		t.Fatalf("allowlist filter output = %#v", output)
	}
	if got := api.callCount("list_projects"); got != 1 {
		t.Fatalf("list projects API calls = %d, want 1", got)
	}

	emptyAPI := newFakeAPI()
	emptyClient := newHarness(t, emptyAPI, security.NewPolicy(false, nil))
	result = callTool(t, emptyClient, "pingcode_list_projects", map[string]any{"page_size": float64(0), "page_index": float64(0)})
	output = resultObject(t, result.StructuredContent)
	if got := output["values"].([]any); len(got) != 0 {
		t.Fatalf("empty allowlist values = %#v, want []", got)
	}
	if output["total"] != float64(0) || output["page_size"] != float64(30) || output["page_index"] != float64(0) {
		t.Fatalf("empty allowlist page output = %#v", output)
	}
	if got := emptyAPI.callCount("list_projects"); got != 0 {
		t.Fatalf("empty allowlist called API %d times, want 0", got)
	}
}

func TestWriteAndProjectAuthorizationRejectBeforeAPI(t *testing.T) {
	api := newFakeAPI()
	client := newHarness(t, api, security.NewPolicy(true, []string{"p1"}))
	for _, tc := range []struct {
		name string
		args map[string]any
		kind string
	}{
		{name: "pingcode_create_work_item", args: map[string]any{"project_id": "p1", "type_id": "task", "title": "x"}, kind: ErrorReadOnly},
		{name: "pingcode_update_work_item", args: map[string]any{"workitem_id": "w1", "project_id": "p1", "title": "x"}, kind: ErrorReadOnly},
		{name: "pingcode_get_project", args: map[string]any{"project_id": "p2"}, kind: ErrorProjectNotAllowed},
		{name: "pingcode_list_work_items", args: map[string]any{"project_id": "p2"}, kind: ErrorProjectNotAllowed},
		{name: "pingcode_get_work_item", args: map[string]any{"workitem_id": "w1", "project_id": "p2"}, kind: ErrorProjectNotAllowed},
		{name: "pingcode_create_work_item", args: map[string]any{"project_id": "p2", "type_id": "task", "title": "x"}, kind: ErrorReadOnly},
	} {
		result := callTool(t, client, tc.name, tc.args)
		if got := errorType(t, result); got != tc.kind {
			t.Errorf("%s error type = %q, want %q", tc.name, got, tc.kind)
		}
	}
	if got := api.callCount("create_work_item") + api.callCount("update_work_item") + api.callCount("get_project") + api.callCount("list_work_items") + api.callCount("get_work_item"); got != 0 {
		t.Fatalf("unauthorized calls reached API: %d", got)
	}

	writeAPI := newFakeAPI()
	writeClient := newHarness(t, writeAPI, security.NewPolicy(false, []string{"p1"}))
	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{name: "pingcode_create_work_item", args: map[string]any{"project_id": "p2", "type_id": "task", "title": "x"}},
		{name: "pingcode_update_work_item", args: map[string]any{"workitem_id": "w1", "project_id": "p2", "title": "x"}},
	} {
		result := callTool(t, writeClient, tc.name, tc.args)
		if got := errorType(t, result); got != ErrorProjectNotAllowed {
			t.Fatalf("unlisted %s type = %q", tc.name, got)
		}
	}
	if writeAPI.callCount("create_work_item")+writeAPI.callCount("update_work_item") != 0 {
		t.Fatal("unlisted write reached API")
	}
}

func TestToolValidationAndPageSizeDefault(t *testing.T) {
	api := newFakeAPI()
	client := newHarness(t, api, security.NewPolicy(false, []string{"p1"}))
	result := callTool(t, client, "pingcode_list_work_items", map[string]any{"project_id": "p1", "page_size": float64(0)})
	if got := resultObject(t, result.StructuredContent)["page_size"]; got != float64(30) {
		t.Fatalf("normalized page_size = %#v, want 30", got)
	}
	if api.callCount("list_work_items") != 1 {
		t.Fatal("normalized request did not reach API")
	}
	api.mu.Lock()
	gotPageSize := api.workParams.PageSize
	api.mu.Unlock()
	if gotPageSize != 30 {
		t.Fatalf("API page_size = %d, want normalized value 30", gotPageSize)
	}

	validationAPI := newFakeAPI()
	validationClient := newHarness(t, validationAPI, security.NewPolicy(false, []string{"p1"}))
	for _, tc := range []struct {
		name string
		args map[string]any
		msg  string
	}{
		{name: "pingcode_list_work_items", args: map[string]any{"project_id": "p1", "page_size": float64(200)}, msg: "page_size"},
		{name: "pingcode_list_projects", args: map[string]any{"page_index": float64(-1)}, msg: "page_index"},
		{name: "pingcode_get_project", args: map[string]any{"project_id": "  "}, msg: "project_id must not be blank"},
		{name: "pingcode_create_work_item", args: map[string]any{"project_id": " ", "type_id": "task", "title": "x"}, msg: "project_id must not be blank"},
		{name: "pingcode_create_work_item", args: map[string]any{"project_id": "p1", "type_id": " \t", "title": "x"}, msg: "type_id must not be blank"},
		{name: "pingcode_create_work_item", args: map[string]any{"project_id": "p1", "type_id": "task", "title": " "}, msg: "title must not be blank"},
		{name: "pingcode_update_work_item", args: map[string]any{"workitem_id": "w1", "project_id": "p1"}, msg: "update requires at least one field"},
	} {
		result := callTool(t, validationClient, tc.name, tc.args)
		if !result.IsError {
			t.Errorf("%s args %#v unexpectedly succeeded", tc.name, tc.args)
			continue
		}
		// SDK schema bounds are checked before the handler; their error is SDK-owned
		// text. All handler-level validations use the structured error envelope.
		if tc.msg == "page_size" || tc.msg == "page_index" {
			if len(result.Content) == 0 || !strings.Contains(textContent(t, result), tc.msg) {
				t.Errorf("SDK schema rejection did not mention %s: %s", tc.msg, textContent(t, result))
			}
			continue
		}
		if got := errorType(t, result); got != ErrorValidation {
			t.Errorf("validation error type = %q", got)
		}
		if !strings.Contains(errorMessage(t, result), tc.msg) {
			t.Errorf("validation message = %q, want substring %q", errorMessage(t, result), tc.msg)
		}
	}
	if got := validationAPI.callCount("get_project") + validationAPI.callCount("create_work_item") + validationAPI.callCount("update_work_item") + validationAPI.callCount("list_projects") + validationAPI.callCount("list_work_items"); got != 0 {
		t.Fatalf("invalid input reached API %d times", got)
	}

	if size, index, err := normalizedPagination(ptrInt(0), ptrInt(4)); err != nil || size != 30 || index != 4 {
		t.Fatalf("normalizedPagination(0, 4) = (%d, %d, %v)", size, index, err)
	}
	for _, tc := range []struct{ size, index int }{{-1, 0}, {101, 0}, {1, -1}} {
		if _, _, err := normalizedPagination(&tc.size, &tc.index); err == nil {
			t.Errorf("normalizedPagination(%d, %d) accepted invalid input", tc.size, tc.index)
		}
	}
}

func TestErrorClassificationAndResponseShape(t *testing.T) {
	const sentinel = "raw-upstream-sensitive-body"
	cases := []struct {
		name, kind, message string
		err                 error
	}{
		{name: "pingcode validation", kind: ErrorValidation, message: "pingcode: VALIDATION", err: &pingcode.Error{Kind: pingcode.KindValidation}},
		{name: "pingcode authentication", kind: ErrorAuthentication, message: "pingcode: AUTHENTICATION", err: &pingcode.Error{Kind: pingcode.KindAuthentication}},
		{name: "pingcode permission", kind: ErrorPermission, message: "pingcode: PERMISSION", err: &pingcode.Error{Kind: pingcode.KindPermission}},
		{name: "pingcode not found", kind: ErrorNotFound, message: "pingcode: NOT_FOUND", err: &pingcode.Error{Kind: pingcode.KindNotFound}},
		{name: "pingcode rate limited", kind: ErrorRateLimited, message: "pingcode: RATE_LIMITED", err: &pingcode.Error{Kind: pingcode.KindRateLimited}},
		{name: "pingcode upstream", kind: ErrorUpstream, message: "pingcode: UPSTREAM", err: &pingcode.Error{Kind: pingcode.KindUpstream}},
		{name: "auth not configured", kind: ErrorAuthentication, message: "client credentials are not configured", err: &auth.Error{Kind: auth.KindNotConfigured}},
		{name: "auth authentication", kind: ErrorAuthentication, message: "token endpoint rejected", err: &auth.Error{Kind: auth.KindAuthentication}},
		{name: "auth upstream", kind: ErrorUpstream, message: "token endpoint request failed", err: &auth.Error{Kind: auth.KindUpstream}},
		{name: "auth invalid response", kind: ErrorUpstream, message: "invalid response", err: &auth.Error{Kind: auth.KindInvalidResponse}},
		{name: "read only", kind: ErrorReadOnly, message: "read-only mode", err: &security.Error{Kind: security.KindReadOnly}},
		{name: "project unauthorized", kind: ErrorProjectNotAllowed, message: "not in the allowlist", err: &security.Error{Kind: security.KindProjectNotAllowed}},
		{name: "deadline", kind: ErrorTimeout, message: "request timed out", err: fmt.Errorf("wrapped: %w", context.DeadlineExceeded)},
		{name: "canceled", kind: ErrorCancelled, message: "request canceled", err: fmt.Errorf("wrapped: %w", context.Canceled)},
		{name: "unknown", kind: ErrorUpstream, message: "internal error", err: errors.New(sentinel)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := newFakeAPI()
			api.err = tc.err
			client := newHarness(t, api, security.NewPolicy(false, []string{"p1"}))
			result := callTool(t, client, "pingcode_get_project", map[string]any{"project_id": "p1"})
			if !result.IsError {
				t.Fatal("error result IsError = false")
			}
			if got := errorType(t, result); got != tc.kind {
				t.Fatalf("error type = %q, want %q", got, tc.kind)
			}
			if got := errorMessage(t, result); !strings.Contains(got, tc.message) {
				t.Fatalf("error message = %q, want substring %q", got, tc.message)
			}
			if tc.name == "unknown" && strings.Contains(textContent(t, result), sentinel) {
				t.Fatal("unknown error leaked original message")
			}
		})
	}
}

func schemaMap(t *testing.T, schema any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	var output map[string]any
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatal(err)
	}
	return output
}

func properties(t *testing.T, schema map[string]any) map[string]any {
	t.Helper()
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties = %#v", schema["properties"])
	}
	return props
}

func assertRequired(t *testing.T, schema map[string]any, names ...string) {
	t.Helper()
	items, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("schema required = %#v", schema["required"])
	}
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item.(string)] = true
	}
	for _, name := range names {
		if !set[name] {
			t.Errorf("required fields %#v omit %q", items, name)
		}
	}
}

func assertHasKeys(t *testing.T, values map[string]any, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, ok := values[name]; !ok {
			t.Errorf("properties omit %q", name)
		}
	}
}

func assertMissingKeys(t *testing.T, values map[string]any, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, ok := values[name]; ok {
			t.Errorf("properties unexpectedly include %q", name)
		}
	}
}

func assertExactKeys(t *testing.T, values map[string]any, names ...string) {
	t.Helper()
	want := make(map[string]bool, len(names))
	for _, name := range names {
		want[name] = true
	}
	for name := range values {
		if !want[name] {
			t.Errorf("unexpected schema property %q", name)
		}
		delete(want, name)
	}
	for name := range want {
		t.Errorf("schema property %q is missing", name)
	}
}

func resultObject(t *testing.T, value any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var output map[string]any
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("structured content is not an object: %v (%s)", err, encoded)
	}
	return output
}

func textContent(t *testing.T, result *sdk.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("tool result has no content")
	}
	text, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("content[0] has type %T, want *mcp.TextContent", result.Content[0])
	}
	return text.Text
}

func errorPayload(t *testing.T, result *sdk.CallToolResult) map[string]any {
	t.Helper()
	var content map[string]any
	if err := json.Unmarshal([]byte(textContent(t, result)), &content); err != nil {
		t.Fatalf("error content is not JSON: %v", err)
	}
	structured := resultObject(t, result.StructuredContent)
	if !reflect.DeepEqual(content, structured) {
		t.Fatalf("error text and structured content differ: %#v vs %#v", content, structured)
	}
	errorObject, ok := content["error"].(map[string]any)
	if !ok || errorObject["type"] == "" || errorObject["message"] == "" {
		t.Fatalf("error payload missing type/message: %#v", content)
	}
	return errorObject
}

func errorType(t *testing.T, result *sdk.CallToolResult) string {
	t.Helper()
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	return errorPayload(t, result)["type"].(string)
}

func errorMessage(t *testing.T, result *sdk.CallToolResult) string {
	t.Helper()
	return errorPayload(t, result)["message"].(string)
}

func ptrInt(value int) *int { return &value }
