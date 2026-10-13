package pingcode

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ihugang/pingcode-mcp-go/internal/auth"
	"github.com/ihugang/pingcode-mcp-go/internal/security"
)

const (
	defaultPageSize = 30
	maxPageSize     = 100
	maxAttempts     = 3
	maxErrorBody    = 64 << 10
)

var fiveXXBackoff = [...]time.Duration{250 * time.Millisecond, 500 * time.Millisecond}

// Option configures a Client.
type Option func(*Client)

// WithSleep injects retry waiting behavior. A nil sleeper is ignored.
func WithSleep(sleep func(ctx context.Context, d time.Duration) error) Option {
	return func(client *Client) {
		if sleep != nil {
			client.sleep = sleep
		}
	}
}

// Client performs contract-bound PingCode REST operations.
type Client struct {
	tokenProvider auth.TokenProvider
	policy        *security.Policy
	httpClient    *http.Client
	baseURL       string
	sleep         func(context.Context, time.Duration) error
}

// New creates a PingCode client. An invalid base URL is reported as an
// UPSTREAM error on the first request, keeping construction simple for callers.
func New(tokenProvider auth.TokenProvider, policy *security.Policy, httpClient *http.Client, baseURL string, options ...Option) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	client := &Client{
		tokenProvider: tokenProvider,
		policy:        policy,
		httpClient:    httpClient,
		baseURL:       strings.TrimRight(baseURL, "/"),
		sleep:         defaultSleep,
	}
	for _, option := range options {
		if option != nil {
			option(client)
		}
	}
	return client
}

// ListProjects is a direct API facade: it does not precheck policy or filter
// results locally. The MCP layer owns allowlist filtering before disclosure.
func (c *Client) ListProjects(ctx context.Context, params ListProjectsParams) (Page[Project], error) {
	var empty Page[Project]
	if blank(params.MemberType) != blank(params.MemberID) {
		return empty, validationError()
	}
	pageSize, pageIndex, err := validatedPagination(params.PageSize, params.PageIndex)
	if err != nil {
		return empty, err
	}
	query := url.Values{}
	setString(query, "scope_type", params.ScopeType)
	setString(query, "scope_id", params.ScopeID)
	setString(query, "keywords", params.Keywords)
	setString(query, "member_type", params.MemberType)
	setString(query, "member_id", params.MemberID)
	setString(query, "type", params.Type)
	setString(query, "created_between", params.CreatedBetween)
	setString(query, "updated_between", params.UpdatedBetween)
	setBool(query, "include_deleted", params.IncludeDeleted)
	setBool(query, "include_archived", params.IncludeArchived)
	setPagination(query, pageSize, pageIndex)

	body, status, err := c.request(ctx, http.MethodGet, "/v1/pjm/projects", query, nil)
	if err != nil {
		return empty, err
	}
	return decodePage(body, status, decodeProject)
}

// GetProject reads one project in an authorized project context.
func (c *Client) GetProject(ctx context.Context, projectID string, options GetOptions) (Project, error) {
	var empty Project
	if blank(projectID) {
		return empty, validationError()
	}
	if err := c.policy.CheckProjectRead(projectID); err != nil {
		return empty, err
	}
	query := getOptionsQuery(options, false)
	body, status, err := c.request(ctx, http.MethodGet, "/v1/pjm/projects/"+url.PathEscape(projectID), query, nil)
	if err != nil {
		return empty, err
	}
	project, err := decodeProject(json.RawMessage(body), status)
	if err != nil {
		return empty, err
	}
	if project.ID != projectID {
		return empty, projectMismatch(projectID)
	}
	return project, nil
}

// ListWorkItems lists work items only within an authorized project context.
func (c *Client) ListWorkItems(ctx context.Context, params ListWorkItemsParams) (Page[WorkItem], error) {
	var empty Page[WorkItem]
	if blank(params.ProjectID) {
		return empty, validationError()
	}
	pageSize, pageIndex, err := validatedPagination(params.PageSize, params.PageIndex)
	if err != nil {
		return empty, err
	}
	if err := c.policy.CheckProjectRead(params.ProjectID); err != nil {
		return empty, err
	}
	query := url.Values{}
	setString(query, "project_id", params.ProjectID)
	setString(query, "identifier", params.Identifier)
	setString(query, "type_id", params.TypeID)
	setString(query, "parent_id", params.ParentID)
	setString(query, "assignee_id", params.AssigneeID)
	setString(query, "state_id", params.StateID)
	setString(query, "priority_id", params.PriorityID)
	setString(query, "bug_type_id", params.BugTypeID)
	setString(query, "tag_id", params.TagID)
	setString(query, "sprint_id", params.SprintID)
	setString(query, "board_id", params.BoardID)
	setString(query, "entry_id", params.EntryID)
	setString(query, "swimlane_id", params.SwimlaneID)
	setString(query, "phase_id", params.PhaseID)
	setString(query, "release_id", params.ReleaseID)
	setString(query, "created_by", params.CreatedBy)
	setString(query, "participant_id", params.ParticipantID)
	setString(query, "keywords", params.Keywords)
	setBool(query, "include_public_image_token", params.IncludePublicImageToken)
	setBool(query, "include_deleted", params.IncludeDeleted)
	setBool(query, "include_archived", params.IncludeArchived)
	setPagination(query, pageSize, pageIndex)

	body, status, err := c.request(ctx, http.MethodGet, "/v1/pjm/workitems", query, nil)
	if err != nil {
		return empty, err
	}
	page, err := decodePage(body, status, decodeWorkItem)
	if err != nil {
		return empty, err
	}
	for _, item := range page.Values {
		if item.Project == nil || item.Project.ID == "" || item.Project.ID != params.ProjectID {
			return empty, projectMismatch(params.ProjectID)
		}
	}
	return page, nil
}

// GetWorkItem reads one work item and verifies its returned project reference.
func (c *Client) GetWorkItem(ctx context.Context, workitemID, projectID string, options GetOptions) (WorkItem, error) {
	var empty WorkItem
	if blank(workitemID) || blank(projectID) {
		return empty, validationError()
	}
	if err := c.policy.CheckProjectRead(projectID); err != nil {
		return empty, err
	}
	query := getOptionsQuery(options, true)
	body, status, err := c.request(ctx, http.MethodGet, "/v1/pjm/workitems/"+url.PathEscape(workitemID), query, nil)
	if err != nil {
		return empty, err
	}
	item, err := decodeWorkItem(json.RawMessage(body), status)
	if err != nil {
		return empty, err
	}
	if item.Project == nil || item.Project.ID == "" || item.Project.ID != projectID {
		return empty, projectMismatch(projectID)
	}
	return item, nil
}

// CreateWorkItem creates a single work item after enforcing client-side write
// policy. The POST request is never retried.
func (c *Client) CreateWorkItem(ctx context.Context, request CreateWorkItemRequest) (WorkItem, error) {
	var empty WorkItem
	if blank(request.ProjectID) || blank(request.TypeID) || blank(request.Title) {
		return empty, validationError()
	}
	if err := c.policy.CheckProjectWrite(request.ProjectID); err != nil {
		return empty, err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return empty, validationError()
	}
	responseBody, status, err := c.request(ctx, http.MethodPost, "/v1/pjm/workitems", nil, body)
	if err != nil {
		return empty, err
	}
	item, err := decodeWorkItem(json.RawMessage(responseBody), status)
	if err != nil {
		return empty, err
	}
	if item.Project == nil || item.Project.ID == "" || item.Project.ID != request.ProjectID {
		return empty, projectMismatch(request.ProjectID)
	}
	return item, nil
}

// UpdateWorkItem verifies project ownership with a GET before PATCH. The PATCH
// request is never retried, so an ambiguous response cannot duplicate a write.
func (c *Client) UpdateWorkItem(ctx context.Context, workitemID, projectID string, update UpdateWorkItemRequest) (WorkItem, error) {
	var empty WorkItem
	if blank(workitemID) || blank(projectID) || !hasUpdateFields(update) {
		return empty, validationError()
	}
	if err := c.policy.CheckProjectWrite(projectID); err != nil {
		return empty, err
	}
	body, err := json.Marshal(update)
	if err != nil {
		return empty, validationError()
	}
	item, err := c.GetWorkItem(ctx, workitemID, projectID, GetOptions{})
	if err != nil {
		return empty, err
	}
	if item.Project == nil || item.Project.ID == "" || item.Project.ID != projectID {
		return empty, projectMismatch(projectID)
	}
	responseBody, status, err := c.request(ctx, http.MethodPatch, "/v1/pjm/workitems/"+url.PathEscape(workitemID), nil, body)
	if err != nil {
		return empty, err
	}
	updated, err := decodeWorkItem(json.RawMessage(responseBody), status)
	if err != nil {
		return empty, err
	}
	if updated.Project == nil || updated.Project.ID == "" || updated.Project.ID != projectID {
		return empty, projectMismatch(projectID)
	}
	return updated, nil
}

func (c *Client) request(ctx context.Context, method, path string, query url.Values, requestBody []byte) ([]byte, int, error) {
	endpoint, err := c.endpoint(path, query)
	if err != nil {
		return nil, 0, err
	}
	if c.tokenProvider == nil {
		return nil, 0, &Error{Kind: KindUpstream}
	}
	token, err := c.tokenProvider.AccessToken(ctx)
	if err != nil {
		return nil, 0, err // Token-provider errors are part of its public contract.
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		request, requestErr := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(requestBody))
		if requestErr != nil {
			return nil, 0, wrapTransportError(nil, requestErr)
		}
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("Accept", "application/json")
		if method == http.MethodPost || method == http.MethodPatch {
			request.Header.Set("Content-Type", "application/json")
		}

		client := *c.httpClient
		// Do not forward a bearer token to a redirect destination.
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
		response, doErr := client.Do(request)
		if doErr != nil {
			return nil, 0, wrapTransportError(ctx.Err(), doErr)
		}

		if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
			responseBody, readErr := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if readErr != nil {
				return nil, 0, wrapTransportError(ctx.Err(), readErr)
			}
			return responseBody, response.StatusCode, nil
		}

		if method == http.MethodGet && response.StatusCode >= 500 && response.StatusCode <= 599 && attempt < maxAttempts {
			_ = discardAndClose(response.Body)
			if sleepErr := c.sleep(ctx, fiveXXBackoff[attempt-1]); sleepErr != nil {
				return nil, 0, sleepErr
			}
			if err := ctx.Err(); err != nil {
				return nil, 0, err
			}
			continue
		}

		responseBody, readErr := readLimitedAndClose(response.Body, maxErrorBody)
		if readErr != nil {
			return nil, 0, wrapTransportError(ctx.Err(), readErr)
		}
		classified := statusError(response.StatusCode, token, responseBody)
		if method == http.MethodGet && response.StatusCode == http.StatusTooManyRequests && attempt < maxAttempts {
			if delay, ok := retryAfter(response.Header); ok {
				if sleepErr := c.sleep(ctx, delay); sleepErr != nil {
					return nil, 0, sleepErr
				}
				if err := ctx.Err(); err != nil {
					return nil, 0, err
				}
				continue
			}
		}
		return nil, 0, classified
	}
	return nil, 0, &Error{Kind: KindUpstream}
}

func (c *Client) endpoint(path string, query url.Values) (*url.URL, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil || !base.IsAbs() || base.Host == "" || base.Opaque != "" || base.User != nil ||
		(!strings.EqualFold(base.Scheme, "http") && !strings.EqualFold(base.Scheme, "https")) ||
		base.RawQuery != "" || base.ForceQuery || base.Fragment != "" {
		return nil, &Error{Kind: KindUpstream}
	}
	endpoint, err := url.Parse(strings.TrimRight(c.baseURL, "/") + path)
	if err != nil {
		return nil, &Error{Kind: KindUpstream}
	}
	endpoint.Scheme = strings.ToLower(endpoint.Scheme)
	endpoint.RawQuery = query.Encode()
	return endpoint, nil
}

func defaultSleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func validatedPagination(pageSize, pageIndex int) (int, int, error) {
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if pageSize < 0 || pageSize > maxPageSize || pageIndex < 0 {
		return 0, 0, validationError()
	}
	return pageSize, pageIndex, nil
}

func setString(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}

func setBool(query url.Values, key string, value bool) {
	if value {
		query.Set(key, "true")
	}
}

func setPagination(query url.Values, pageSize, pageIndex int) {
	query.Set("page_size", strconv.Itoa(pageSize))
	query.Set("page_index", strconv.Itoa(pageIndex))
}

func getOptionsQuery(options GetOptions, workItem bool) url.Values {
	query := url.Values{}
	setBool(query, "include_deleted", options.IncludeDeleted)
	setBool(query, "include_archived", options.IncludeArchived)
	if workItem {
		setBool(query, "include_public_image_token", options.IncludePublicImageToken)
	}
	return query
}

func decodeProject(raw json.RawMessage, status int) (Project, error) {
	var project Project
	if err := json.Unmarshal(raw, &project); err != nil || project.ID == "" {
		return Project{}, &Error{Kind: KindUpstream, Status: status}
	}
	project.Raw = append(json.RawMessage(nil), raw...)
	return project, nil
}

func decodeWorkItem(raw json.RawMessage, status int) (WorkItem, error) {
	var item WorkItem
	if err := json.Unmarshal(raw, &item); err != nil || item.ID == "" {
		return WorkItem{}, &Error{Kind: KindUpstream, Status: status}
	}
	item.Raw = append(json.RawMessage(nil), raw...)
	return item, nil
}

func decodePage[T any](body []byte, status int, decode func(json.RawMessage, int) (T, error)) (Page[T], error) {
	var source struct {
		PageSize  *int            `json:"page_size"`
		PageIndex *int            `json:"page_index"`
		Total     *int            `json:"total"`
		Values    json.RawMessage `json:"values"`
	}
	if err := json.Unmarshal(body, &source); err != nil || source.PageSize == nil || source.PageIndex == nil ||
		source.Total == nil || len(source.Values) == 0 || string(bytes.TrimSpace(source.Values)) == "null" {
		return Page[T]{}, &Error{Kind: KindUpstream, Status: status}
	}
	var values []json.RawMessage
	if err := json.Unmarshal(source.Values, &values); err != nil {
		return Page[T]{}, &Error{Kind: KindUpstream, Status: status}
	}
	page := Page[T]{PageSize: *source.PageSize, PageIndex: *source.PageIndex, Total: *source.Total}
	page.Values = make([]T, 0, len(values))
	for _, raw := range values {
		value, err := decode(raw, status)
		if err != nil {
			return Page[T]{}, err
		}
		page.Values = append(page.Values, value)
	}
	return page, nil
}

func readLimitedAndClose(reader io.ReadCloser, limit int64) ([]byte, error) {
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, limit))
}

func discardAndClose(reader io.ReadCloser) error {
	defer reader.Close()
	_, err := io.Copy(io.Discard, io.LimitReader(reader, maxErrorBody))
	return err
}

func retryAfter(header http.Header) (time.Duration, bool) {
	// The headers are interpreted as seconds per docs/api-contract.md §3;
	// tenant-specific units and formatting remain INTEGRATION-PENDING.
	for _, key := range []string{"X-RateLimit-Retry-After", "X-PC-Retry-After"} {
		value := strings.TrimSpace(header.Get(key))
		if value == "" {
			continue
		}
		seconds, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 ||
			seconds >= float64(math.MaxInt64)/float64(time.Second) {
			continue
		}
		return time.Duration(seconds * float64(time.Second)), true
	}
	return 0, false
}

func blank(value string) bool { return strings.TrimSpace(value) == "" }

func projectMismatch(projectID string) error {
	return &security.Error{Kind: security.KindProjectNotAllowed, ProjectID: projectID}
}

func hasUpdateFields(update UpdateWorkItemRequest) bool {
	return update.Title != nil || update.Description != nil || update.StartAt != nil || update.EndAt != nil ||
		update.PriorityID != nil || update.StateID != nil || update.AssigneeID != nil || update.ParentID != nil ||
		len(update.ReleaseIDs) != 0 || update.BoardID != nil || update.EntryID != nil || update.SwimlaneID != nil ||
		update.PhaseID != nil || update.StoryPoints != nil || update.EstimatedWorkload != nil ||
		update.RemainingWorkload != nil || len(update.Properties) != 0
}
