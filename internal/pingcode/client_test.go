package pingcode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ihugang/pingcode-mcp-go/internal/security"
)

// Tests here trace to bootstrap-pingcode-mcp tasks 2.3/2.4, capability
// pingcode-rest-client, secret-redaction, and PRD FR-004 through FR-011.
// PingCode demand ID: PENDING. All HTTP interactions use httptest or a local
// RoundTripper; injected sleepers ensure no test performs real retry waits.

const testToken = "test-access-token-987"

const liveTenantWorkItemJSON = `{
	"id": "wi-live-1",
	"url": "https://example.invalid/v1/pjm/workitems/wi-live-1",
	"project": {"id": "project-1", "url": "https://example.invalid/v1/pjm/projects/project-1", "name": "Live Project", "type": "scrum", "identifier": "LIVE", "is_archived": 0, "is_deleted": 0},
	"identifier": "LIVE-1",
	"title": "Live item",
	"type": "task",
	"start_at": null,
	"end_at": null,
	"parent_id": null,
	"short_id": "4RJpEHc7",
	"html_url": "https://example.invalid/pjm/workitems/4RJpEHc7",
	"parent": null,
	"assignee": null,
	"state": {"id": "state-1", "url": "https://example.invalid/v1/pjm/workitem_states/state-1", "name": "Open", "type": "pending", "color": "#56ABFB"},
	"priority": null,
	"board": null,
	"entry": null,
	"swimlane": null,
	"release": null,
	"releases": null,
	"sprint": null,
	"phase": null,
	"story_points": null,
	"estimated_workload": null,
	"remaining_workload": null,
	"description": null,
	"completed_at": null,
	"properties": {"job_type": null},
	"entity_properties": {},
	"tags": [],
	"participants": [],
	"created_at": 1791534887,
	"created_by": {"id": "user-0", "url": "https://example.invalid/v1/directory/users/user-0", "display_name": "Operator", "name": "operator"},
	"updated_at": 1791534887,
	"updated_by": {"id": "user-0", "url": "https://example.invalid/v1/directory/users/user-0", "display_name": "Operator", "name": "operator"}
}`

type staticTokenProvider struct {
	token string
	err   error
}

func (p staticTokenProvider) AccessToken(context.Context) (string, error) {
	return p.token, p.err
}

func testPolicy(readOnly bool, projectIDs ...string) *security.Policy {
	return security.NewPolicy(readOnly, projectIDs)
}

func testClient(server *httptest.Server, policy *security.Policy, waits *[]time.Duration) *Client {
	sleeper := func(ctx context.Context, delay time.Duration) error {
		if waits != nil {
			*waits = append(*waits, delay)
		}
		return ctx.Err()
	}
	return New(staticTokenProvider{token: testToken}, policy, server.Client(), server.URL+"/", WithSleep(sleeper))
}

func assertPingcodeError(t *testing.T, err error, kind ErrorKind) *Error {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s error", kind)
	}
	var pingcodeErr *Error
	if !errors.As(err, &pingcodeErr) {
		t.Fatalf("expected *pingcode.Error, got %T: %v", err, err)
	}
	if pingcodeErr.Kind != kind {
		t.Fatalf("error kind = %q, want %q (error: %v)", pingcodeErr.Kind, kind, err)
	}
	return pingcodeErr
}

func assertSecurityError(t *testing.T, err error, kind security.ViolationKind) *security.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("expected security error %s", kind)
	}
	var policyErr *security.Error
	if !errors.As(err, &policyErr) {
		t.Fatalf("expected *security.Error, got %T: %v", err, err)
	}
	if policyErr.Kind != kind {
		t.Fatalf("security error kind = %q, want %q", policyErr.Kind, kind)
	}
	return policyErr
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func assertRequestHeaders(t *testing.T, request *http.Request, contentType bool) {
	t.Helper()
	if got := request.Header.Get("Authorization"); got != "Bearer "+testToken {
		t.Errorf("Authorization = %q", got)
	}
	if got := request.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q", got)
	}
	if contentType {
		if got := request.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
	} else if got := request.Header.Get("Content-Type"); got != "" {
		t.Errorf("unexpected Content-Type on GET: %q", got)
	}
}

func assertQuery(t *testing.T, got url.Values, want url.Values) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("query = %#v, want %#v", got, want)
	}
}

func decodeObject(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return object
}

func TestListProjectsContractAndRawMapping(t *testing.T) {
	var waits []time.Duration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/v1/pjm/projects" {
			t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
		}
		assertRequestHeaders(t, r, false)
		want := url.Values{
			"scope_type": {"team"}, "scope_id": {"scope-1"}, "keywords": {"project words"},
			"member_type": {"user"}, "member_id": {"member-1"}, "type": {"scrum"},
			"created_between": {"1,2"}, "updated_between": {"3,4"},
			"include_deleted": {"true"}, "include_archived": {"true"},
			"page_size": {"17"}, "page_index": {"3"},
		}
		assertQuery(t, r.URL.Query(), want)
		writeJSON(w, http.StatusOK, `{"page_size":17,"page_index":3,"total":2,"values":[{"id":"proj-1","url":"https://example.invalid/p/1","identifier":"PC","name":"Project One","type":"scrum","state":{"id":"active","name":"Active","type":"state"},"assignee":{"id":"user-1","name":"A User","type":"user"},"created_at":11,"updated_at":12,"contract_extension":{"kept":true}},{"id":"proj-2","name":"Project Two"}]}`)
	}))
	defer server.Close()
	client := testClient(server, testPolicy(false, "proj-1"), &waits)

	page, err := client.ListProjects(context.Background(), ListProjectsParams{
		ScopeType: "team", ScopeID: "scope-1", Keywords: "project words", MemberType: "user", MemberID: "member-1",
		Type: "scrum", CreatedBetween: "1,2", UpdatedBetween: "3,4", IncludeDeleted: true,
		IncludeArchived: true, PageSize: 17, PageIndex: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.PageSize != 17 || page.PageIndex != 3 || page.Total != 2 || len(page.Values) != 2 {
		t.Fatalf("unexpected page metadata: %#v", page)
	}
	project := page.Values[0]
	if project.ID != "proj-1" || project.URL == "" || project.Identifier != "PC" || project.Name != "Project One" || project.Type != "scrum" ||
		project.State == nil || project.State.ID != "active" || project.Assignee == nil || project.Assignee.ID != "user-1" ||
		project.CreatedAt != 11 || project.UpdatedAt != 12 {
		t.Fatalf("project field mapping failed: %#v", project)
	}
	if !strings.Contains(string(project.Raw), `"contract_extension":{"kept":true}`) {
		t.Fatalf("Raw did not retain unmodeled fields: %s", project.Raw)
	}
	if page.Values[1].ID != "proj-2" {
		t.Fatalf("project order changed: %#v", page.Values)
	}
	if len(waits) != 0 {
		t.Fatalf("unexpected waits: %v", waits)
	}
}

func TestGetProjectContractAndEscapedPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/v1/pjm/projects/team%2Falpha" {
			t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
		}
		assertRequestHeaders(t, r, false)
		assertQuery(t, r.URL.Query(), url.Values{"include_deleted": {"true"}, "include_archived": {"true"}})
		writeJSON(w, http.StatusOK, `{"id":"team/alpha","name":"Team Alpha","state":{"id":"active"},"created_at":1,"updated_at":2}`)
	}))
	defer server.Close()
	client := testClient(server, testPolicy(false, "team/alpha"), nil)

	project, err := client.GetProject(context.Background(), "team/alpha", GetOptions{IncludeDeleted: true, IncludeArchived: true})
	if err != nil {
		t.Fatal(err)
	}
	if project.ID != "team/alpha" || project.State == nil || project.State.ID != "active" || len(project.Raw) == 0 {
		t.Fatalf("unexpected project: %#v", project)
	}
}

func TestListWorkItemsContractAndMapping(t *testing.T) {
	var waits []time.Duration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/v1/pjm/workitems" {
			t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
		}
		assertRequestHeaders(t, r, false)
		want := url.Values{
			"project_id": {"proj-1"}, "identifier": {"P-4"}, "type_id": {"story"}, "parent_id": {"parent"},
			"assignee_id": {"assignee"}, "state_id": {"active"}, "priority_id": {"p1"}, "bug_type_id": {"bug"},
			"tag_id": {"tag"}, "sprint_id": {"sprint"}, "board_id": {"board"}, "entry_id": {"entry"},
			"swimlane_id": {"swimlane"}, "phase_id": {"phase"}, "release_id": {"release"}, "created_by": {"creator"},
			"participant_id": {"participant"}, "keywords": {"work words"}, "include_public_image_token": {"true"},
			"include_deleted": {"true"}, "include_archived": {"true"}, "page_size": {"23"}, "page_index": {"2"},
		}
		assertQuery(t, r.URL.Query(), want)
		writeJSON(w, http.StatusOK, `{"page_size":23,"page_index":2,"total":1,"values":[{"id":"wi-1","identifier":"P-4","title":"Fix thing","description":"details","type":"story","state":{"id":"active","name":"Active"},"priority":{"id":"p1","name":"High"},"assignee":{"id":"user-1"},"project":{"id":"proj-1","name":"Project One","type":"project"},"parent":{"id":"parent-1","name":"Parent"},"created_at":21,"updated_at":22,"attributes":{"custom":"kept"}}]}`)
	}))
	defer server.Close()
	client := testClient(server, testPolicy(false, "proj-1"), &waits)

	page, err := client.ListWorkItems(context.Background(), ListWorkItemsParams{
		ProjectID: "proj-1", Identifier: "P-4", TypeID: "story", ParentID: "parent", AssigneeID: "assignee",
		StateID: "active", PriorityID: "p1", BugTypeID: "bug", TagID: "tag", SprintID: "sprint", BoardID: "board",
		EntryID: "entry", SwimlaneID: "swimlane", PhaseID: "phase", ReleaseID: "release", CreatedBy: "creator",
		ParticipantID: "participant", Keywords: "work words", IncludePublicImageToken: true, IncludeDeleted: true,
		IncludeArchived: true, PageSize: 23, PageIndex: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.PageSize != 23 || page.PageIndex != 2 || page.Total != 1 || len(page.Values) != 1 {
		t.Fatalf("unexpected page metadata: %#v", page)
	}
	item := page.Values[0]
	if item.ID != "wi-1" || item.Identifier != "P-4" || item.Title != "Fix thing" || item.Description != "details" ||
		item.Type != "story" || item.State == nil || item.State.ID != "active" ||
		item.Priority == nil || item.Priority.ID != "p1" || item.Assignee == nil || item.Assignee.ID != "user-1" ||
		item.Project == nil || item.Project.ID != "proj-1" || item.Parent == nil || item.Parent.ID != "parent-1" ||
		item.CreatedAt != 21 || item.UpdatedAt != 22 || len(item.Raw) == 0 {
		t.Fatalf("work-item field mapping failed: %#v", item)
	}
}

// Regression coverage for FR-012, bootstrap-pingcode-mcp task 5.3 / M8-1:
// a live-tenant response exposed that work-item type is a string enum.
func TestWorkItemDecodeMatchesLiveTenantShape(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/v1/pjm/workitems/wi-live-1" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			writeJSON(w, http.StatusOK, liveTenantWorkItemJSON)
		}))
		defer server.Close()

		client := testClient(server, testPolicy(false, "project-1"), nil)
		item, err := client.GetWorkItem(context.Background(), "wi-live-1", "project-1", GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if item.Type != "task" || item.Project == nil || item.Project.ID != "project-1" || item.State == nil || item.State.ID != "state-1" {
			t.Fatalf("unexpected decoded item: %#v", item)
		}
	})

	t.Run("list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/v1/pjm/workitems" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			writeJSON(w, http.StatusOK, `{"page_size":30,"page_index":0,"total":1,"values":[`+liveTenantWorkItemJSON+`]}`)
		}))
		defer server.Close()

		client := testClient(server, testPolicy(false, "project-1"), nil)
		page, err := client.ListWorkItems(context.Background(), ListWorkItemsParams{ProjectID: "project-1"})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Values) != 1 || page.Values[0].Type != "task" {
			t.Fatalf("unexpected list result: %#v", page)
		}
	})

	t.Run("create", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/v1/pjm/workitems" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			writeJSON(w, http.StatusOK, liveTenantWorkItemJSON)
		}))
		defer server.Close()

		client := testClient(server, testPolicy(false, "project-1"), nil)
		item, err := client.CreateWorkItem(context.Background(), CreateWorkItemRequest{ProjectID: "project-1", TypeID: "task", Title: "Live item"})
		if err != nil {
			t.Fatal(err)
		}
		if item.Type != "task" || item.ID == "" {
			t.Fatalf("unexpected created item: %#v", item)
		}
	})

	t.Run("update", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				if r.URL.Path != "/v1/pjm/workitems/wi-live-1" {
					t.Errorf("GET path = %s", r.URL.Path)
				}
				writeJSON(w, http.StatusOK, liveTenantWorkItemJSON)
			case http.MethodPatch:
				if r.URL.Path != "/v1/pjm/workitems/wi-live-1" {
					t.Errorf("PATCH path = %s", r.URL.Path)
				}
				writeJSON(w, http.StatusOK, liveTenantWorkItemJSON)
			default:
				t.Errorf("unexpected method %s", r.Method)
				http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
			}
		}))
		defer server.Close()

		client := testClient(server, testPolicy(false, "project-1"), nil)
		updated, err := client.UpdateWorkItem(context.Background(), "wi-live-1", "project-1", UpdateWorkItemRequest{Title: ptr("Updated title")})
		if err != nil {
			t.Fatal(err)
		}
		if updated.Type != "task" {
			t.Fatalf("updated type = %q, want task", updated.Type)
		}
	})
}

func TestGetWorkItemContractAndEscapedPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/v1/pjm/workitems/item%2Falpha" {
			t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
		}
		assertRequestHeaders(t, r, false)
		assertQuery(t, r.URL.Query(), url.Values{
			"include_deleted": {"true"}, "include_archived": {"true"}, "include_public_image_token": {"true"},
		})
		writeJSON(w, http.StatusOK, `{"id":"item/alpha","identifier":"P-1","title":"A title","project":{"id":"proj-1"}}`)
	}))
	defer server.Close()
	client := testClient(server, testPolicy(false, "proj-1"), nil)

	item, err := client.GetWorkItem(context.Background(), "item/alpha", "proj-1", GetOptions{
		IncludeDeleted: true, IncludeArchived: true, IncludePublicImageToken: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.ID != "item/alpha" || item.Project == nil || item.Project.ID != "proj-1" {
		t.Fatalf("unexpected item: %#v", item)
	}
}

func TestCreateWorkItemContractBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.EscapedPath() != "/v1/pjm/workitems" {
			t.Errorf("request = %s %s", r.Method, r.URL.EscapedPath())
		}
		assertRequestHeaders(t, r, true)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		object := decodeObject(t, body)
		want := map[string]any{
			"project_id": "proj-1", "type_id": "bug", "title": "Create title", "description": "description",
			"start_at": float64(10), "end_at": float64(20), "priority_id": "p1", "state_id": "open",
			"assignee_id": "user-1", "parent_id": "parent-1", "sprint_id": "sprint-1", "release_ids": []any{"release-1"},
			"board_id": "board-1", "entry_id": "entry-1", "swimlane_id": "swimlane-1", "story_points": float64(3.5),
			"estimated_workload": float64(5), "remaining_workload": float64(2), "properties": map[string]any{"custom": "value"},
			"participant_ids": []any{"user-2"},
		}
		if len(object) != len(want) {
			t.Errorf("create body fields = %v, want %d fields", keys(object), len(want))
		}
		for key, expected := range want {
			var actual any
			if err := json.Unmarshal(object[key], &actual); err != nil {
				t.Errorf("decode body field %s: %v", key, err)
				continue
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("body[%q] = %#v, want %#v", key, actual, expected)
			}
		}
		writeJSON(w, http.StatusCreated, `{"id":"wi-created","project":{"id":"proj-1"}}`)
	}))
	defer server.Close()
	client := testClient(server, testPolicy(false, "proj-1"), nil)

	request := CreateWorkItemRequest{
		ProjectID: "proj-1", TypeID: "bug", Title: "Create title", Description: ptr("description"),
		StartAt: ptr(int64(10)), EndAt: ptr(int64(20)), PriorityID: ptr("p1"), StateID: ptr("open"),
		AssigneeID: ptr("user-1"), ParentID: ptr("parent-1"), SprintID: ptr("sprint-1"), ReleaseIDs: []string{"release-1"},
		BoardID: ptr("board-1"), EntryID: ptr("entry-1"), SwimlaneID: ptr("swimlane-1"), StoryPoints: ptr(3.5),
		EstimatedWorkload: ptr(5.0), RemainingWorkload: ptr(2.0), Properties: map[string]any{"custom": "value"},
		ParticipantIDs: []string{"user-2"},
	}
	item, err := client.CreateWorkItem(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if item.ID != "wi-created" || item.Project == nil || item.Project.ID != "proj-1" {
		t.Fatalf("unexpected created item: %#v", item)
	}
}

func TestCreateWorkItemOmitsUnsetOptionalFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		object := decodeObject(t, body)
		if len(object) != 3 {
			t.Errorf("minimal create body fields = %v, want only required fields", keys(object))
		}
		for _, key := range []string{"project_id", "type_id", "title"} {
			if _, ok := object[key]; !ok {
				t.Errorf("create body missing required field %q", key)
			}
		}
		writeJSON(w, http.StatusCreated, `{"id":"wi-created","project":{"id":"proj-1"}}`)
	}))
	defer server.Close()
	client := testClient(server, testPolicy(false, "proj-1"), nil)
	_, err := client.CreateWorkItem(context.Background(), CreateWorkItemRequest{ProjectID: "proj-1", TypeID: "bug", Title: "title"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateWorkItemContractBodyAndPreRead(t *testing.T) {
	var getCount, patchCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/v1/pjm/workitems/wi-1" {
			t.Errorf("path = %s", r.URL.EscapedPath())
		}
		switch r.Method {
		case http.MethodGet:
			getCount.Add(1)
			assertRequestHeaders(t, r, false)
			writeJSON(w, http.StatusOK, `{"id":"wi-1","project":{"id":"proj-1"}}`)
		case http.MethodPatch:
			patchCount.Add(1)
			assertRequestHeaders(t, r, true)
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
			}
			object := decodeObject(t, body)
			if len(object) != 17 {
				t.Errorf("PATCH fields = %v, want all 17 allowed fields", keys(object))
			}
			for _, key := range []string{"title", "description", "start_at", "end_at", "priority_id", "state_id", "assignee_id", "parent_id", "release_ids", "board_id", "entry_id", "swimlane_id", "phase_id", "story_points", "estimated_workload", "remaining_workload", "properties"} {
				if _, ok := object[key]; !ok {
					t.Errorf("PATCH omitted explicitly set field %q", key)
				}
			}
			for _, forbidden := range []string{"sprint_id", "participant_ids", "project_id", "type_id"} {
				if _, ok := object[forbidden]; ok {
					t.Errorf("PATCH included forbidden field %q", forbidden)
				}
			}
			writeJSON(w, http.StatusOK, `{"id":"wi-1","project":{"id":"proj-1"}}`)
		default:
			t.Errorf("unexpected method %s", r.Method)
			http.Error(w, "unexpected", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	client := testClient(server, testPolicy(false, "proj-1"), nil)
	update := UpdateWorkItemRequest{
		Title: ptr("updated"), Description: ptr("new details"), StartAt: ptr(int64(1)), EndAt: ptr(int64(2)),
		PriorityID: ptr("p2"), StateID: ptr("closed"), AssigneeID: ptr("user-2"), ParentID: ptr("parent-2"),
		ReleaseIDs: []string{"release-2"}, BoardID: ptr("board-2"), EntryID: ptr("entry-2"), SwimlaneID: ptr("swimlane-2"),
		PhaseID: ptr("phase-2"), StoryPoints: ptr(8.0), EstimatedWorkload: ptr(13.0), RemainingWorkload: ptr(3.0),
		Properties: map[string]any{"new": true},
	}
	item, err := client.UpdateWorkItem(context.Background(), "wi-1", "proj-1", update)
	if err != nil {
		t.Fatal(err)
	}
	if item.ID != "wi-1" || getCount.Load() != 1 || patchCount.Load() != 1 {
		t.Fatalf("result=%#v GET=%d PATCH=%d", item, getCount.Load(), patchCount.Load())
	}
}

func TestUpdateWorkItemSendsOnlySetFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, `{"id":"wi-1","project":{"id":"proj-1"}}`)
			return
		}
		if r.Method != http.MethodPatch {
			t.Errorf("unexpected method %s", r.Method)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if string(body) != `{"title":"only this"}` {
			t.Errorf("PATCH body = %s, want only title", body)
		}
		writeJSON(w, http.StatusOK, `{"id":"wi-1","project":{"id":"proj-1"}}`)
	}))
	defer server.Close()
	client := testClient(server, testPolicy(false, "proj-1"), nil)
	_, err := client.UpdateWorkItem(context.Background(), "wi-1", "proj-1", UpdateWorkItemRequest{Title: ptr("only this")})
	if err != nil {
		t.Fatal(err)
	}
}

func TestListProjectsIsUnfilteredFacade(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		writeJSON(w, http.StatusOK, `{"page_size":30,"page_index":0,"total":1,"values":[{"id":"outside"}]}`)
	}))
	defer server.Close()
	client := testClient(server, testPolicy(false), nil)
	page, err := client.ListProjects(context.Background(), ListProjectsParams{})
	if err != nil {
		t.Fatal(err)
	}
	if count.Load() != 1 || len(page.Values) != 1 || page.Values[0].ID != "outside" {
		t.Fatalf("ListProjects unexpectedly enforced policy/filtering: %#v, count=%d", page, count.Load())
	}
}

func TestPaginationDefaultsBoundsAndValidation(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		switch r.URL.EscapedPath() {
		case "/v1/pjm/projects":
			assertQuery(t, r.URL.Query(), url.Values{"page_size": {"30"}, "page_index": {"0"}})
			writeJSON(w, http.StatusOK, `{"page_size":30,"page_index":0,"total":0,"values":[]}`)
		case "/v1/pjm/workitems":
			assertQuery(t, r.URL.Query(), url.Values{"project_id": {"proj-1"}, "page_size": {"30"}, "page_index": {"0"}})
			writeJSON(w, http.StatusOK, `{"page_size":30,"page_index":0,"total":0,"values":[]}`)
		default:
			t.Errorf("unexpected list path %s", r.URL.EscapedPath())
		}
	}))
	defer server.Close()
	client := testClient(server, testPolicy(false, "proj-1"), nil)
	if _, err := client.ListProjects(context.Background(), ListProjectsParams{}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListWorkItems(context.Background(), ListWorkItemsParams{ProjectID: "proj-1"}); err != nil {
		t.Fatal(err)
	}
	if count.Load() != 2 {
		t.Fatalf("successful list requests = %d, want 2", count.Load())
	}

	invalid := []struct {
		name string
		call func() error
	}{
		{"projects page_size over max", func() error {
			_, err := client.ListProjects(context.Background(), ListProjectsParams{PageSize: 101})
			return err
		}},
		{"projects negative page_size", func() error {
			_, err := client.ListProjects(context.Background(), ListProjectsParams{PageSize: -1})
			return err
		}},
		{"projects negative page_index", func() error {
			_, err := client.ListProjects(context.Background(), ListProjectsParams{PageIndex: -1})
			return err
		}},
		{"workitems page_size over max", func() error {
			_, err := client.ListWorkItems(context.Background(), ListWorkItemsParams{ProjectID: "proj-1", PageSize: 101})
			return err
		}},
		{"workitems negative page_size", func() error {
			_, err := client.ListWorkItems(context.Background(), ListWorkItemsParams{ProjectID: "proj-1", PageSize: -1})
			return err
		}},
		{"workitems negative page_index", func() error {
			_, err := client.ListWorkItems(context.Background(), ListWorkItemsParams{ProjectID: "proj-1", PageIndex: -1})
			return err
		}},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			assertPingcodeError(t, test.call(), KindValidation)
		})
	}
	if count.Load() != 2 {
		t.Fatalf("invalid pagination sent requests: count=%d", count.Load())
	}
}

func TestLocalValidationRejectsBeforeHTTP(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count.Add(1)
		writeJSON(w, http.StatusOK, `{}`)
	}))
	defer server.Close()
	client := testClient(server, testPolicy(false, "proj-1"), nil)

	checks := []struct {
		name string
		call func() error
	}{
		{"member type without member id", func() error {
			_, err := client.ListProjects(context.Background(), ListProjectsParams{MemberType: "user"})
			return err
		}},
		{"member id without member type", func() error {
			_, err := client.ListProjects(context.Background(), ListProjectsParams{MemberID: "user-1"})
			return err
		}},
		{"work item list without project", func() error { _, err := client.ListWorkItems(context.Background(), ListWorkItemsParams{}); return err }},
		{"create without project", func() error {
			_, err := client.CreateWorkItem(context.Background(), CreateWorkItemRequest{TypeID: "bug", Title: "title"})
			return err
		}},
		{"create without type", func() error {
			_, err := client.CreateWorkItem(context.Background(), CreateWorkItemRequest{ProjectID: "proj-1", Title: "title"})
			return err
		}},
		{"create without title", func() error {
			_, err := client.CreateWorkItem(context.Background(), CreateWorkItemRequest{ProjectID: "proj-1", TypeID: "bug"})
			return err
		}},
		{"update without fields", func() error {
			_, err := client.UpdateWorkItem(context.Background(), "wi-1", "proj-1", UpdateWorkItemRequest{})
			return err
		}},
		{"update without item id", func() error {
			_, err := client.UpdateWorkItem(context.Background(), "", "proj-1", UpdateWorkItemRequest{Title: ptr("title")})
			return err
		}},
		{"update without project id", func() error {
			_, err := client.UpdateWorkItem(context.Background(), "wi-1", "", UpdateWorkItemRequest{Title: ptr("title")})
			return err
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) { assertPingcodeError(t, check.call(), KindValidation) })
	}
	if count.Load() != 0 {
		t.Fatalf("validation failures sent %d HTTP requests", count.Load())
	}
}

func TestPolicyChecksRejectBeforeRequests(t *testing.T) {
	var readOnlyCount atomic.Int32
	readOnlyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		readOnlyCount.Add(1)
		writeJSON(w, http.StatusOK, `{}`)
	}))
	defer readOnlyServer.Close()
	readOnlyClient := testClient(readOnlyServer, testPolicy(true, "proj-1"), nil)
	assertSecurityError(t, createForTest(readOnlyClient), security.KindReadOnly)
	_, updateErr := readOnlyClient.UpdateWorkItem(context.Background(), "wi-1", "proj-1", UpdateWorkItemRequest{Title: ptr("updated")})
	assertSecurityError(t, updateErr, security.KindReadOnly)
	if readOnlyCount.Load() != 0 {
		t.Fatalf("read-only calls sent %d HTTP requests", readOnlyCount.Load())
	}

	var outsideCount atomic.Int32
	outsideServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		outsideCount.Add(1)
		writeJSON(w, http.StatusOK, `{}`)
	}))
	defer outsideServer.Close()
	outsideClient := testClient(outsideServer, testPolicy(false, "proj-allowed"), nil)
	checks := []struct {
		name string
		call func() error
	}{
		{"get project", func() error {
			_, err := outsideClient.GetProject(context.Background(), "proj-denied", GetOptions{})
			return err
		}},
		{"list work items", func() error {
			_, err := outsideClient.ListWorkItems(context.Background(), ListWorkItemsParams{ProjectID: "proj-denied"})
			return err
		}},
		{"get work item", func() error {
			_, err := outsideClient.GetWorkItem(context.Background(), "wi-1", "proj-denied", GetOptions{})
			return err
		}},
		{"create", func() error {
			_, err := outsideClient.CreateWorkItem(context.Background(), CreateWorkItemRequest{ProjectID: "proj-denied", TypeID: "bug", Title: "title"})
			return err
		}},
		{"update", func() error {
			_, err := outsideClient.UpdateWorkItem(context.Background(), "wi-1", "proj-denied", UpdateWorkItemRequest{Title: ptr("title")})
			return err
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			assertSecurityError(t, check.call(), security.KindProjectNotAllowed)
		})
	}
	if outsideCount.Load() != 0 {
		t.Fatalf("out-of-allowlist calls sent %d HTTP requests", outsideCount.Load())
	}
}

func createForTest(client *Client) error {
	_, err := client.CreateWorkItem(context.Background(), CreateWorkItemRequest{ProjectID: "proj-1", TypeID: "bug", Title: "title"})
	return err
}

func TestProjectOwnershipFailuresReturnNoResource(t *testing.T) {
	t.Run("project detail", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, `{"id":"proj-other","name":"must not return"}`)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false, "proj-1"), nil)
		project, err := client.GetProject(context.Background(), "proj-1", GetOptions{})
		assertSecurityError(t, err, security.KindProjectNotAllowed)
		if project.ID != "" {
			t.Fatalf("mismatched project data returned: %#v", project)
		}
	})

	t.Run("work item detail", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, `{"id":"wi-1","title":"must not return","project":{"id":"proj-other"}}`)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false, "proj-1"), nil)
		item, err := client.GetWorkItem(context.Background(), "wi-1", "proj-1", GetOptions{})
		assertSecurityError(t, err, security.KindProjectNotAllowed)
		if item.ID != "" {
			t.Fatalf("mismatched work item data returned: %#v", item)
		}
	})

	t.Run("work item list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, `{"page_size":30,"page_index":0,"total":1,"values":[{"id":"wi-1","title":"must not return","project":{"id":"proj-other"}}]}`)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false, "proj-1"), nil)
		page, err := client.ListWorkItems(context.Background(), ListWorkItemsParams{ProjectID: "proj-1"})
		assertSecurityError(t, err, security.KindProjectNotAllowed)
		if len(page.Values) != 0 {
			t.Fatalf("mismatched work item page returned: %#v", page)
		}
	})

	t.Run("update pre-read blocks patch", func(t *testing.T) {
		var gets, patches atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				gets.Add(1)
				writeJSON(w, http.StatusOK, `{"id":"wi-1","project":{"id":"proj-other"}}`)
				return
			}
			if r.Method == http.MethodPatch {
				patches.Add(1)
			}
			writeJSON(w, http.StatusOK, `{}`)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false, "proj-1"), nil)
		item, err := client.UpdateWorkItem(context.Background(), "wi-1", "proj-1", UpdateWorkItemRequest{Title: ptr("changed")})
		assertSecurityError(t, err, security.KindProjectNotAllowed)
		if item.ID != "" || gets.Load() != 1 || patches.Load() != 0 {
			t.Fatalf("item=%#v GET=%d PATCH=%d", item, gets.Load(), patches.Load())
		}
	})

	t.Run("create response project mismatch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusCreated, `{"id":"wi-1","project":{"id":"proj-other"}}`)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false, "proj-1"), nil)
		item, err := client.CreateWorkItem(context.Background(), CreateWorkItemRequest{ProjectID: "proj-1", TypeID: "bug", Title: "title"})
		assertSecurityError(t, err, security.KindProjectNotAllowed)
		if item.ID != "" {
			t.Fatalf("mismatched created item returned: %#v", item)
		}
	})
}

func TestHTTPErrorClassificationAndRedaction(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		kind   ErrorKind
		code   string
	}{
		{"401", http.StatusUnauthorized, KindAuthentication, "100001"},
		{"403", http.StatusForbidden, KindPermission, "100002"},
		{"404", http.StatusNotFound, KindNotFound, "100006"},
		{"500", http.StatusInternalServerError, KindUpstream, "100000"},
		{"other with envelope", http.StatusBadRequest, KindUpstream, "100000"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, test.status, `{"code":"`+test.code+`","message":"response-body-marker constructed-client-secret `+testToken+`"}`)
			}))
			defer server.Close()
			client := testClient(server, testPolicy(false), nil)
			_, err := client.ListProjects(context.Background(), ListProjectsParams{Keywords: "query-secret-value"})
			pingcodeErr := assertPingcodeError(t, err, test.kind)
			if pingcodeErr.Status != test.status || pingcodeErr.APICode != test.code {
				t.Fatalf("status/code = %d/%q, want %d/%q", pingcodeErr.Status, pingcodeErr.APICode, test.status, test.code)
			}
			for _, forbidden := range []string{"response-body-marker", "constructed-client-secret", testToken, "query-secret-value", server.URL} {
				if strings.Contains(err.Error(), forbidden) {
					t.Errorf("error text exposed %q: %s", forbidden, err)
				}
			}
		})
	}
}

func TestErrorCodeDropsAuthorizationLikeOrTokenValues(t *testing.T) {
	for _, code := range []string{"Bearer-test-token", "authorization_header", testToken} {
		t.Run(code, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusBadRequest, `{"code":"`+code+`","message":"hidden"}`)
			}))
			defer server.Close()
			client := testClient(server, testPolicy(false), nil)
			_, err := client.ListProjects(context.Background(), ListProjectsParams{})
			pingcodeErr := assertPingcodeError(t, err, KindUpstream)
			if pingcodeErr.APICode != "" || strings.Contains(strings.ToLower(err.Error()), strings.ToLower(code)) {
				t.Fatalf("unsafe code leaked: %#v / %s", pingcodeErr, err)
			}
		})
	}
}

func TestGetRateLimitRetryHeadersAndPrecedence(t *testing.T) {
	for _, test := range []struct {
		name    string
		headers map[string]string
		want    []time.Duration
	}{
		{"public header", map[string]string{"X-RateLimit-Retry-After": "2"}, []time.Duration{2 * time.Second}},
		{"private decimal header", map[string]string{"X-PC-Retry-After": "1.5"}, []time.Duration{1500 * time.Millisecond}},
		{"public header takes precedence", map[string]string{"X-RateLimit-Retry-After": "2", "X-PC-Retry-After": "1.5"}, []time.Duration{2 * time.Second}},
		{"invalid public falls back to private", map[string]string{"X-RateLimit-Retry-After": "invalid", "X-PC-Retry-After": "0.25"}, []time.Duration{250 * time.Millisecond}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			var waits []time.Duration
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if requests.Add(1) == 1 {
					for name, value := range test.headers {
						w.Header().Set(name, value)
					}
					writeJSON(w, http.StatusTooManyRequests, `{"code":"100029","message":"limited"}`)
					return
				}
				writeJSON(w, http.StatusOK, `{"page_size":30,"page_index":0,"total":0,"values":[]}`)
			}))
			defer server.Close()
			client := testClient(server, testPolicy(false), &waits)
			if _, err := client.ListProjects(context.Background(), ListProjectsParams{}); err != nil {
				t.Fatal(err)
			}
			if requests.Load() != 2 || !reflect.DeepEqual(waits, test.want) {
				t.Fatalf("requests=%d waits=%v, want 2 and %v", requests.Load(), waits, test.want)
			}
		})
	}
}

func TestGetRateLimitMissingHeaderAndExhaustion(t *testing.T) {
	t.Run("missing header fails immediately", func(t *testing.T) {
		var requests atomic.Int32
		var waits []time.Duration
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			writeJSON(w, http.StatusTooManyRequests, `{"code":"100029","message":"limited"}`)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false), &waits)
		_, err := client.ListProjects(context.Background(), ListProjectsParams{})
		pingcodeErr := assertPingcodeError(t, err, KindRateLimited)
		if pingcodeErr.Status != http.StatusTooManyRequests || requests.Load() != 1 || len(waits) != 0 {
			t.Fatalf("status=%d requests=%d waits=%v", pingcodeErr.Status, requests.Load(), waits)
		}
	})

	t.Run("invalid headers fail immediately", func(t *testing.T) {
		var requests atomic.Int32
		var waits []time.Duration
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.Header().Set("X-RateLimit-Retry-After", "NaN")
			w.Header().Set("X-PC-Retry-After", "-1")
			writeJSON(w, http.StatusTooManyRequests, `{"message":"limited"}`)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false), &waits)
		_, err := client.ListProjects(context.Background(), ListProjectsParams{})
		assertPingcodeError(t, err, KindRateLimited)
		if requests.Load() != 1 || len(waits) != 0 {
			t.Fatalf("requests=%d waits=%v, want 1 and no waits", requests.Load(), waits)
		}
	})

	t.Run("three attempts exhaust budget", func(t *testing.T) {
		var requests atomic.Int32
		var waits []time.Duration
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.Header().Set("X-RateLimit-Retry-After", "0.1")
			writeJSON(w, http.StatusTooManyRequests, `{"code":"100029","message":"limited"}`)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false), &waits)
		_, err := client.ListProjects(context.Background(), ListProjectsParams{})
		assertPingcodeError(t, err, KindRateLimited)
		wantWaits := []time.Duration{100 * time.Millisecond, 100 * time.Millisecond}
		if requests.Load() != 3 || !reflect.DeepEqual(waits, wantWaits) {
			t.Fatalf("requests=%d waits=%v, want 3 and %v", requests.Load(), waits, wantWaits)
		}
	})
}

func TestGetFiveXXBackoffAndExhaustion(t *testing.T) {
	t.Run("recovers after bounded backoff", func(t *testing.T) {
		var requests atomic.Int32
		var waits []time.Duration
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if requests.Add(1) < 3 {
				writeJSON(w, http.StatusServiceUnavailable, `{"message":"temporary"}`)
				return
			}
			writeJSON(w, http.StatusOK, `{"page_size":30,"page_index":0,"total":0,"values":[]}`)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false), &waits)
		if _, err := client.ListProjects(context.Background(), ListProjectsParams{}); err != nil {
			t.Fatal(err)
		}
		want := []time.Duration{250 * time.Millisecond, 500 * time.Millisecond}
		if requests.Load() != 3 || !reflect.DeepEqual(waits, want) {
			t.Fatalf("requests=%d waits=%v, want 3 and %v", requests.Load(), waits, want)
		}
	})

	t.Run("returns last status when exhausted", func(t *testing.T) {
		var requests atomic.Int32
		var waits []time.Duration
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			writeJSON(w, http.StatusBadGateway, `{"code":"500001","message":"temporary"}`)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false), &waits)
		_, err := client.ListProjects(context.Background(), ListProjectsParams{})
		pingcodeErr := assertPingcodeError(t, err, KindUpstream)
		want := []time.Duration{250 * time.Millisecond, 500 * time.Millisecond}
		if pingcodeErr.Status != http.StatusBadGateway || requests.Load() != 3 || !reflect.DeepEqual(waits, want) {
			t.Fatalf("status=%d requests=%d waits=%v", pingcodeErr.Status, requests.Load(), waits)
		}
	})
}

func TestWritesAreNeverReplayed(t *testing.T) {
	t.Run("POST 429", func(t *testing.T) {
		var requests atomic.Int32
		var waits []time.Duration
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.Header().Set("X-RateLimit-Retry-After", "1")
			writeJSON(w, http.StatusTooManyRequests, `{"code":"100029","message":"limited"}`)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false, "proj-1"), &waits)
		_, err := client.CreateWorkItem(context.Background(), CreateWorkItemRequest{ProjectID: "proj-1", TypeID: "bug", Title: "title"})
		assertPingcodeError(t, err, KindRateLimited)
		if requests.Load() != 1 || len(waits) != 0 { // Actual assertion: the POST is sent once and never replayed.
			t.Fatalf("POST requests=%d waits=%v, want 1 and no waits", requests.Load(), waits)
		}
	})

	t.Run("PATCH 5xx after one pre-read", func(t *testing.T) {
		var getCount, patchCount atomic.Int32
		var waits []time.Duration
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				getCount.Add(1)
				writeJSON(w, http.StatusOK, `{"id":"wi-1","project":{"id":"proj-1"}}`)
				return
			}
			if r.Method == http.MethodPatch {
				patchCount.Add(1)
				writeJSON(w, http.StatusInternalServerError, `{"message":"failed"}`)
				return
			}
			t.Errorf("unexpected method %s", r.Method)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false, "proj-1"), &waits)
		_, err := client.UpdateWorkItem(context.Background(), "wi-1", "proj-1", UpdateWorkItemRequest{Title: ptr("updated")})
		assertPingcodeError(t, err, KindUpstream)
		if getCount.Load() != 1 || patchCount.Load() != 1 || len(waits) != 0 { // Actual assertion: PATCH occurs once; only the required GET precedes it.
			t.Fatalf("GET=%d PATCH=%d waits=%v", getCount.Load(), patchCount.Load(), waits)
		}
	})
}

func TestRetryWaitCancellationStopsFurtherAttempts(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writeJSON(w, http.StatusServiceUnavailable, `{"message":"retry"}`)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := New(staticTokenProvider{token: testToken}, testPolicy(false), server.Client(), server.URL,
		WithSleep(func(ctx context.Context, _ time.Duration) error {
			cancel()
			return ctx.Err()
		}),
	)
	_, err := client.ListProjects(ctx, ListProjectsParams{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("requests after cancellation = %d, want 1", requests.Load())
	}
}

func TestCancellationAfterRetrySleeperReturnsStopsFurtherAttempts(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writeJSON(w, http.StatusServiceUnavailable, `{"message":"retry"}`)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := New(staticTokenProvider{token: testToken}, testPolicy(false), server.Client(), server.URL,
		WithSleep(func(context.Context, time.Duration) error {
			cancel()
			return nil
		}),
	)
	_, err := client.ListProjects(ctx, ListProjectsParams{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("requests after cancellation = %d, want 1", requests.Load())
	}
}

func TestHTTPRequestCancellationPropagates(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := testClient(server, testPolicy(false), nil)
	result := make(chan error, 1)
	go func() {
		_, err := client.ListProjects(ctx, ListProjectsParams{})
		result <- err
	}()
	<-started
	cancel()
	err := <-result
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestTransportAndBaseURLErrorsAreSafe(t *testing.T) {
	t.Run("transport failure", func(t *testing.T) {
		sentinel := errors.New("dial failure")
		transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("%w at %s?query-secret-value=%s", sentinel, request.URL.String(), testToken)
		})
		client := New(staticTokenProvider{token: testToken}, testPolicy(false), &http.Client{Transport: transport}, "https://api.example")
		_, err := client.ListProjects(context.Background(), ListProjectsParams{Keywords: "query-secret-value"})
		assertPingcodeError(t, err, KindUpstream)
		if !errors.Is(err, sentinel) {
			t.Fatalf("transport cause was not preserved: %v", err)
		}
		for _, secret := range []string{"api.example", "query-secret-value", testToken} {
			if strings.Contains(err.Error(), secret) {
				t.Errorf("error exposed %q: %v", secret, err)
			}
		}
	})

	t.Run("timeout cause remains identifiable", func(t *testing.T) {
		transport := roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		})
		client := New(staticTokenProvider{token: testToken}, testPolicy(false), &http.Client{Transport: transport}, "https://api.example")
		_, err := client.ListProjects(context.Background(), ListProjectsParams{})
		assertPingcodeError(t, err, KindUpstream)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("timeout cause was not preserved: %v", err)
		}
	})

	t.Run("invalid base URL", func(t *testing.T) {
		client := New(staticTokenProvider{token: testToken}, testPolicy(false), http.DefaultClient, "https://api.example?client_secret=constructed-client-secret")
		_, err := client.ListProjects(context.Background(), ListProjectsParams{})
		assertPingcodeError(t, err, KindUpstream)
		if strings.Contains(err.Error(), "constructed-client-secret") || strings.Contains(err.Error(), "api.example") {
			t.Fatalf("base URL leaked in error: %v", err)
		}
	})
}

func TestTokenProviderErrorPassesThroughUnchanged(t *testing.T) {
	sentinel := errors.New("safe token-provider sentinel")
	client := New(staticTokenProvider{err: sentinel}, testPolicy(false), http.DefaultClient, "https://example.invalid")
	_, err := client.ListProjects(context.Background(), ListProjectsParams{})
	if err != sentinel {
		t.Fatalf("token-provider error = %v, want identical error %v", err, sentinel)
	}
}

func TestInvalidSuccessResponsesAreUpstreamErrors(t *testing.T) {
	for _, body := range []string{"not-json", `{}`, `{"page_size":30,"page_index":0,"total":1,"values":[{"name":"missing id"}]}`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusOK, body)
			}))
			defer server.Close()
			client := testClient(server, testPolicy(false), nil)
			_, err := client.ListProjects(context.Background(), ListProjectsParams{})
			pingcodeErr := assertPingcodeError(t, err, KindUpstream)
			if pingcodeErr.Status != http.StatusOK {
				t.Fatalf("decode error status=%d, want 200", pingcodeErr.Status)
			}
			if strings.Contains(err.Error(), body) {
				t.Fatalf("invalid response body leaked: %v", err)
			}
		})
	}

	t.Run("detail missing id", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, `{"name":"missing id"}`)
		}))
		defer server.Close()
		client := testClient(server, testPolicy(false, "proj-1"), nil)
		_, err := client.GetProject(context.Background(), "proj-1", GetOptions{})
		assertPingcodeError(t, err, KindUpstream)
	})
}

func ptr[T any](value T) *T { return &value }

func keys(object map[string]json.RawMessage) []string {
	result := make([]string, 0, len(object))
	for key := range object {
		result = append(result, key)
	}
	return result
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
