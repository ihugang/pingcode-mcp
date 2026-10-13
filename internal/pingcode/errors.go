package pingcode

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

// ErrorKind is a stable category for PingCode client errors.
type ErrorKind string

const (
	KindValidation     ErrorKind = "VALIDATION"
	KindAuthentication ErrorKind = "AUTHENTICATION"
	KindPermission     ErrorKind = "PERMISSION"
	KindNotFound       ErrorKind = "NOT_FOUND"
	KindRateLimited    ErrorKind = "RATE_LIMITED"
	KindUpstream       ErrorKind = "UPSTREAM"
)

// Error exposes only safe classification metadata. The cause is kept private
// and wrapped so transport URLs remain hidden while errors.Is still works.
type Error struct {
	Kind    ErrorKind
	Status  int
	APICode string

	cause error
}

// Error returns stable text that never includes response bodies, URLs, or
// transport error details.
func (e *Error) Error() string {
	if e == nil {
		return "pingcode: UPSTREAM"
	}

	kind := e.Kind
	switch kind {
	case KindValidation, KindAuthentication, KindPermission, KindNotFound, KindRateLimited, KindUpstream:
	default:
		kind = KindUpstream
	}

	message := "pingcode: " + string(kind)
	if e.Status > 0 {
		message += " (HTTP " + itoa(e.Status)
		if code := safeAPICode(e.APICode, ""); code != "" {
			message += ", code " + code
		}
		message += ")"
	}
	return message
}

// Unwrap returns a safe cause that still supports errors.Is matching.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// safeCause hides the original transport message, which may contain a URL,
// while retaining its identity for errors.Is and errors.As.
type safeCause struct {
	err error
}

func (e *safeCause) Error() string { return "pingcode: request failed" }
func (e *safeCause) Unwrap() error { return e.err }

func validationError() error {
	return &Error{Kind: KindValidation}
}

func safeAPICode(code, accessToken string) string {
	if len(code) == 0 || len(code) > 64 {
		return ""
	}
	lower := strings.ToLower(code)
	if strings.Contains(lower, "bearer") || strings.Contains(lower, "authorization") {
		return ""
	}
	if accessToken != "" && strings.Contains(lower, strings.ToLower(accessToken)) {
		return ""
	}
	for _, r := range code {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.') {
			return ""
		}
	}
	return code
}

func errorCodeFromBody(body []byte, accessToken string) string {
	var envelope struct {
		Code json.RawMessage `json:"code"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || len(envelope.Code) == 0 {
		return ""
	}

	var text string
	if err := json.Unmarshal(envelope.Code, &text); err == nil {
		return safeAPICode(text, accessToken)
	}
	if json.Valid(envelope.Code) {
		text = string(envelope.Code)
		if text != "null" {
			return safeAPICode(text, accessToken)
		}
	}
	return ""
}

func statusError(status int, accessToken string, body []byte) *Error {
	kind := KindUpstream
	switch status {
	case 401:
		kind = KindAuthentication
	case 403:
		kind = KindPermission
	case 404:
		kind = KindNotFound
	case 429:
		kind = KindRateLimited
	}
	return &Error{Kind: kind, Status: status, APICode: errorCodeFromBody(body, accessToken)}
}

func wrapTransportError(ctxErr, transportErr error) error {
	if ctxErr != nil {
		return &Error{Kind: KindUpstream, cause: &safeCause{err: ctxErr}}
	}
	if transportErr != nil {
		return &Error{Kind: KindUpstream, cause: &safeCause{err: transportErr}}
	}
	return &Error{Kind: KindUpstream, cause: errors.New("pingcode: request failed")}
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
