// Package pingcode implements the PingCode REST API client, including contract
// models, pagination, error classification, bounded GET retries, and client-side
// write and project-allowlist checks. ListProjects intentionally remains a
// global API facade without policy prechecks or local filtering; the MCP layer
// is responsible for filtering projects before returning them to callers.
//
// Implementation tracking: bootstrap-pingcode-mcp tasks 2.3, 2.4, and the
// client-layer portion of 3.1; PRD FR-004 through FR-010 and FR-011;
// PingCode demand ID: PENDING.
package pingcode
