// Package mcp implementation tracking: bootstrap-pingcode-mcp tasks 4.1–4.2;
// PRD FR-003–FR-009; PingCode demand ID: PENDING.
package mcp

import (
	"context"

	"github.com/ihugang/pingcode-mcp-go/internal/pingcode"
)

// PingCodeAPI is the narrow REST surface used by the P0 MCP tools.
type PingCodeAPI interface {
	ListProjects(ctx context.Context, params pingcode.ListProjectsParams) (pingcode.Page[pingcode.Project], error)
	GetProject(ctx context.Context, projectID string, opts pingcode.GetOptions) (pingcode.Project, error)
	ListWorkItems(ctx context.Context, params pingcode.ListWorkItemsParams) (pingcode.Page[pingcode.WorkItem], error)
	GetWorkItem(ctx context.Context, workitemID, projectID string, opts pingcode.GetOptions) (pingcode.WorkItem, error)
	CreateWorkItem(ctx context.Context, req pingcode.CreateWorkItemRequest) (pingcode.WorkItem, error)
	UpdateWorkItem(ctx context.Context, workitemID, projectID string, req pingcode.UpdateWorkItemRequest) (pingcode.WorkItem, error)
}

var _ PingCodeAPI = (*pingcode.Client)(nil)
