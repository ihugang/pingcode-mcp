package pingcode

import "encoding/json"

// Reference is the compact representation used for related PingCode resources.
type Reference struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// Project contains the contract fields returned by project endpoints. Raw
// retains the complete upstream object, including fields not modeled here.
type Project struct {
	ID         string          `json:"id"`
	URL        string          `json:"url"`
	Identifier string          `json:"identifier"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	State      *Reference      `json:"state"`
	Assignee   *Reference      `json:"assignee"`
	CreatedAt  int64           `json:"created_at"`
	UpdatedAt  int64           `json:"updated_at"`
	Raw        json.RawMessage `json:"-"`
}

// WorkItem contains the contract fields returned by work-item endpoints. Raw
// retains the complete upstream object, including fields not modeled here.
type WorkItem struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	// Type is the system work-item type string observed in live tenant create/get/list responses (for example, "task"; 2026-10-09).
	Type      string          `json:"type"`
	State     *Reference      `json:"state"`
	Priority  *Reference      `json:"priority"`
	Assignee  *Reference      `json:"assignee"`
	Project   *Reference      `json:"project"`
	Parent    *Reference      `json:"parent"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
	Raw       json.RawMessage `json:"-"`
}

// Page preserves the pagination envelope returned by PingCode.
type Page[T any] struct {
	PageSize  int `json:"page_size"`
	PageIndex int `json:"page_index"`
	Total     int `json:"total"`
	Values    []T `json:"values"`
}

// GetOptions controls optional fields on project and work-item detail reads.
type GetOptions struct {
	IncludeDeleted          bool
	IncludeArchived         bool
	IncludePublicImageToken bool
}

// ListProjectsParams contains the supported project-list filters.
type ListProjectsParams struct {
	ScopeType       string
	ScopeID         string
	Keywords        string
	MemberType      string
	MemberID        string
	Type            string
	CreatedBetween  string
	UpdatedBetween  string
	IncludeDeleted  bool
	IncludeArchived bool
	PageSize        int
	PageIndex       int
}

// ListWorkItemsParams contains the supported work-item-list filters.
type ListWorkItemsParams struct {
	ProjectID               string
	Identifier              string
	TypeID                  string
	ParentID                string
	AssigneeID              string
	StateID                 string
	PriorityID              string
	BugTypeID               string
	TagID                   string
	SprintID                string
	BoardID                 string
	EntryID                 string
	SwimlaneID              string
	PhaseID                 string
	ReleaseID               string
	CreatedBy               string
	ParticipantID           string
	Keywords                string
	IncludePublicImageToken bool
	IncludeDeleted          bool
	IncludeArchived         bool
	PageSize                int
	PageIndex               int
}

// CreateWorkItemRequest contains the fields allowed by contract §5.5.
type CreateWorkItemRequest struct {
	ProjectID         string         `json:"project_id"`
	TypeID            string         `json:"type_id"`
	Title             string         `json:"title"`
	Description       *string        `json:"description,omitempty"`
	StartAt           *int64         `json:"start_at,omitempty"`
	EndAt             *int64         `json:"end_at,omitempty"`
	PriorityID        *string        `json:"priority_id,omitempty"`
	StateID           *string        `json:"state_id,omitempty"`
	AssigneeID        *string        `json:"assignee_id,omitempty"`
	ParentID          *string        `json:"parent_id,omitempty"`
	SprintID          *string        `json:"sprint_id,omitempty"`
	ReleaseIDs        []string       `json:"release_ids,omitempty"`
	BoardID           *string        `json:"board_id,omitempty"`
	EntryID           *string        `json:"entry_id,omitempty"`
	SwimlaneID        *string        `json:"swimlane_id,omitempty"`
	StoryPoints       *float64       `json:"story_points,omitempty"`
	EstimatedWorkload *float64       `json:"estimated_workload,omitempty"`
	RemainingWorkload *float64       `json:"remaining_workload,omitempty"`
	Properties        map[string]any `json:"properties,omitempty"`
	ParticipantIDs    []string       `json:"participant_ids,omitempty"`
}

// UpdateWorkItemRequest contains only the fields allowed by contract §5.6.
type UpdateWorkItemRequest struct {
	Title             *string        `json:"title,omitempty"`
	Description       *string        `json:"description,omitempty"`
	StartAt           *int64         `json:"start_at,omitempty"`
	EndAt             *int64         `json:"end_at,omitempty"`
	PriorityID        *string        `json:"priority_id,omitempty"`
	StateID           *string        `json:"state_id,omitempty"`
	AssigneeID        *string        `json:"assignee_id,omitempty"`
	ParentID          *string        `json:"parent_id,omitempty"`
	ReleaseIDs        []string       `json:"release_ids,omitempty"`
	BoardID           *string        `json:"board_id,omitempty"`
	EntryID           *string        `json:"entry_id,omitempty"`
	SwimlaneID        *string        `json:"swimlane_id,omitempty"`
	PhaseID           *string        `json:"phase_id,omitempty"`
	StoryPoints       *float64       `json:"story_points,omitempty"`
	EstimatedWorkload *float64       `json:"estimated_workload,omitempty"`
	RemainingWorkload *float64       `json:"remaining_workload,omitempty"`
	Properties        map[string]any `json:"properties,omitempty"`
}
