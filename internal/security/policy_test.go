// These tests cover the security primitives for OpenSpec change
// bootstrap-pingcode-mcp task 3.1 and PRD FR-010. PingCode requirement ID:
// PENDING.
package security

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

func TestAllowsProject(t *testing.T) {
	tests := []struct {
		name      string
		allowlist []string
		projectID string
		want      bool
	}{
		{name: "exact match", allowlist: []string{"project-1"}, projectID: "project-1", want: true},
		{name: "case-sensitive mismatch", allowlist: []string{"Project-1"}, projectID: "project-1"},
		{name: "leading whitespace is not normalized", allowlist: []string{"project-1"}, projectID: " project-1"},
		{name: "trailing whitespace is not normalized", allowlist: []string{"project-1"}, projectID: "project-1 "},
		{name: "whitespace in allowlist is preserved exactly", allowlist: []string{" project-1 "}, projectID: " project-1 ", want: true},
		{name: "empty allowlist", projectID: "project-1"},
		{name: "empty project ID", allowlist: []string{""}, projectID: ""},
		{name: "whitespace-only project ID", allowlist: []string{"   "}, projectID: "   "},
		{name: "multiple projects", allowlist: []string{"project-1", "project-2"}, projectID: "project-2", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := NewPolicy(false, tt.allowlist)
			if got := policy.AllowsProject(tt.projectID); got != tt.want {
				t.Fatalf("AllowsProject(%q) = %t, want %t", tt.projectID, got, tt.want)
			}
		})
	}
}

func TestAllowsAnyProject(t *testing.T) {
	tests := []struct {
		name   string
		policy *Policy
		want   bool
	}{
		{name: "nil policy"},
		{name: "empty allowlist", policy: NewPolicy(false, nil)},
		{name: "non-empty allowlist", policy: NewPolicy(false, []string{"project-1"}), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.policy.AllowsAnyProject(); got != tt.want {
				t.Fatalf("AllowsAnyProject() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestCheckProjectRead(t *testing.T) {
	tests := []struct {
		name      string
		allowlist []string
		projectID string
		wantKind  ViolationKind
	}{
		{name: "allowlisted project", allowlist: []string{"project-1"}, projectID: "project-1"},
		{name: "project not allowlisted", allowlist: []string{"project-1"}, projectID: "project-2", wantKind: KindProjectNotAllowed},
		{name: "empty allowlist", projectID: "project-1", wantKind: KindProjectNotAllowed},
		{name: "empty project ID", allowlist: []string{"project-1"}, wantKind: KindProjectNotAllowed},
		{name: "whitespace project ID", allowlist: []string{"project-1"}, projectID: " \t ", wantKind: KindProjectNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewPolicy(false, tt.allowlist).CheckProjectRead(tt.projectID)
			assertPolicyError(t, err, tt.wantKind)
		})
	}
}

func TestCheckProjectWrite(t *testing.T) {
	tests := []struct {
		name      string
		readOnly  bool
		allowlist []string
		projectID string
		wantKind  ViolationKind
	}{
		{name: "read-write mode and allowlisted project", allowlist: []string{"project-1"}, projectID: "project-1"},
		{name: "read-only mode rejects allowlisted project", readOnly: true, allowlist: []string{"project-1"}, projectID: "project-1", wantKind: KindReadOnly},
		{name: "read-write mode rejects unlisted project", allowlist: []string{"project-1"}, projectID: "project-2", wantKind: KindProjectNotAllowed},
		{name: "read-write mode rejects empty allowlist", projectID: "project-1", wantKind: KindProjectNotAllowed},
		{name: "read-only takes precedence for unlisted project", readOnly: true, allowlist: []string{"project-1"}, projectID: "project-2", wantKind: KindReadOnly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewPolicy(tt.readOnly, tt.allowlist).CheckProjectWrite(tt.projectID)
			assertPolicyError(t, err, tt.wantKind)
		})
	}
}

func TestIsReadOnly(t *testing.T) {
	tests := []struct {
		name     string
		readOnly bool
	}{
		{name: "enabled", readOnly: true},
		{name: "disabled", readOnly: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewPolicy(tt.readOnly, nil).IsReadOnly(); got != tt.readOnly {
				t.Fatalf("IsReadOnly() = %t, want %t", got, tt.readOnly)
			}
		})
	}
}

func TestNewPolicyCopiesAllowlist(t *testing.T) {
	allowlist := []string{"project-1"}
	policy := NewPolicy(false, allowlist)
	allowlist[0] = "project-2"
	allowlist = append(allowlist, "project-3")

	if !policy.AllowsProject("project-1") {
		t.Fatal("policy lost its original project after the caller changed the slice")
	}
	for _, projectID := range []string{"project-2", "project-3"} {
		if policy.AllowsProject(projectID) {
			t.Errorf("policy unexpectedly allows caller-added project %q", projectID)
		}
	}
}

func TestPolicyErrorIsStableAndDoesNotEchoInput(t *testing.T) {
	const projectID = "credential-like-sentinel"
	err := NewPolicy(false, nil).CheckProjectRead(projectID)

	var violation *Error
	if !errors.As(err, &violation) {
		t.Fatalf("errors.As(%T, **security.Error) = false", err)
	}
	if violation.ProjectID != projectID {
		t.Fatalf("ProjectID = %q, want %q", violation.ProjectID, projectID)
	}
	if strings.Contains(err.Error(), projectID) {
		t.Fatalf("policy error exposed caller input: %q", err)
	}
	if got, want := err.Error(), "security: project is not in the allowlist"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestRedactSecrets(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		secrets []string
		want    string
	}{
		{name: "ordinary replacement", text: "token=alpha", secrets: []string{"alpha"}, want: "token=" + Redacted},
		{name: "multiple occurrences", text: "alpha alpha alpha", secrets: []string{"alpha"}, want: Redacted + " " + Redacted + " " + Redacted},
		{
			name:    "query-escaped replacement",
			text:    "raw=+/= encoded=" + url.QueryEscape("+/="),
			secrets: []string{"+/="},
			want:    "raw=" + Redacted + " encoded=" + Redacted,
		},
		{name: "multiple secrets", text: "id=alpha&secret=bravo", secrets: []string{"alpha", "bravo"}, want: "id=" + Redacted + "&secret=" + Redacted},
		{name: "empty secret leaves text alone", text: "unchanged", secrets: []string{""}, want: "unchanged"},
		{name: "empty text is safe", secrets: []string{"secret"}, want: ""},
		{name: "empty text and empty secret are safe", secrets: []string{""}, want: ""},
		{name: "longest overlapping secret", text: "abcd abc", secrets: []string{"abc", "abcd"}, want: Redacted + " " + Redacted},
		{name: "replacement text is not rescanned", text: "secret", secrets: []string{"secret", Redacted}, want: Redacted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactSecrets(tt.text, tt.secrets...); got != tt.want {
				t.Fatalf("RedactSecrets(%q, %q...) = %q, want %q", tt.text, tt.secrets, got, tt.want)
			}
		})
	}
}

func assertPolicyError(t *testing.T, err error, wantKind ViolationKind) {
	t.Helper()
	if wantKind == "" {
		if err != nil {
			t.Fatalf("got error %v, want nil", err)
		}
		return
	}

	var violation *Error
	if !errors.As(err, &violation) {
		t.Fatalf("errors.As(%T, **security.Error) = false", err)
	}
	if violation.Kind != wantKind {
		t.Fatalf("error kind = %q, want %q", violation.Kind, wantKind)
	}
}
