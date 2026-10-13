package security

import (
	"net/url"
	"strings"
)

// ViolationKind identifies a locally enforced security policy violation.
type ViolationKind string

const (
	KindReadOnly          ViolationKind = "READ_ONLY"
	KindProjectNotAllowed ViolationKind = "PROJECT_NOT_ALLOWED"
)

// Error reports a stable, safe policy error. ProjectID is retained for callers
// that need non-sensitive authorization context for troubleshooting.
type Error struct {
	Kind      ViolationKind
	ProjectID string
}

// Error returns a stable message without echoing caller-supplied values.
func (e *Error) Error() string {
	if e == nil {
		return "security: policy violation"
	}

	switch e.Kind {
	case KindReadOnly:
		return "security: read-only mode rejects write operations"
	case KindProjectNotAllowed:
		return "security: project is not in the allowlist"
	default:
		return "security: policy violation"
	}
}

// Policy enforces project-level read authorization and the global write gate.
type Policy struct {
	readOnly          bool
	allowedProjectIDs []string
}

// NewPolicy builds a policy from configuration values without retaining the
// caller's allowlist slice. readOnly enables a global write gate, and an empty
// allowedProjectIDs list authorizes no project.
func NewPolicy(readOnly bool, allowedProjectIDs []string) *Policy {
	projectIDs := append([]string(nil), allowedProjectIDs...)
	return &Policy{
		readOnly:          readOnly,
		allowedProjectIDs: projectIDs,
	}
}

// IsReadOnly reports whether the global write gate is enabled. A nil policy
// fails closed and is treated as read-only.
func (p *Policy) IsReadOnly() bool {
	return p == nil || p.readOnly
}

// AllowsProject reports whether projectID exactly matches an allowlist entry.
// Empty and whitespace-only IDs are never authorized.
func (p *Policy) AllowsProject(projectID string) bool {
	if p == nil || strings.TrimSpace(projectID) == "" {
		return false
	}

	for _, allowedProjectID := range p.allowedProjectIDs {
		if projectID == allowedProjectID {
			return true
		}
	}
	return false
}

// AllowsAnyProject reports whether the policy contains at least one project
// in its allowlist. A nil policy fails closed.
func (p *Policy) AllowsAnyProject() bool {
	return p != nil && len(p.allowedProjectIDs) > 0
}

// CheckProjectRead validates the project authorization context for a targeted
// read operation.
func (p *Policy) CheckProjectRead(projectID string) error {
	if !p.AllowsProject(projectID) {
		return &Error{Kind: KindProjectNotAllowed, ProjectID: projectID}
	}
	return nil
}

// CheckProjectWrite validates the global write gate and then project access.
// Read-only mode takes precedence over project authorization failures.
func (p *Policy) CheckProjectWrite(projectID string) error {
	if p.IsReadOnly() {
		return &Error{Kind: KindReadOnly, ProjectID: projectID}
	}
	if !p.AllowsProject(projectID) {
		return &Error{Kind: KindProjectNotAllowed, ProjectID: projectID}
	}
	return nil
}

// Redacted is the replacement text used by RedactSecrets.
const Redacted = "[REDACTED]"

// RedactSecrets replaces each non-empty secret and its URL query-escaped form
// with Redacted. It scans the original text once so inserted replacement text
// cannot be processed as though it were part of the input.
func RedactSecrets(text string, secrets ...string) string {
	if text == "" || len(secrets) == 0 {
		return text
	}

	patterns := make([]string, 0, len(secrets)*2)
	seen := make(map[string]struct{}, len(secrets)*2)
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		for _, pattern := range []string{secret, url.QueryEscape(secret)} {
			if _, ok := seen[pattern]; ok {
				continue
			}
			seen[pattern] = struct{}{}
			patterns = append(patterns, pattern)
		}
	}
	if len(patterns) == 0 {
		return text
	}

	var result strings.Builder
	result.Grow(len(text))
	for offset := 0; offset < len(text); {
		matchStart := len(text)
		matchPattern := ""
		for _, pattern := range patterns {
			index := strings.Index(text[offset:], pattern)
			if index < 0 {
				continue
			}
			start := offset + index
			if start < matchStart || (start == matchStart && len(pattern) > len(matchPattern)) {
				matchStart = start
				matchPattern = pattern
			}
		}
		if matchPattern == "" {
			result.WriteString(text[offset:])
			break
		}

		result.WriteString(text[offset:matchStart])
		result.WriteString(Redacted)
		offset = matchStart + len(matchPattern)
	}
	return result.String()
}
