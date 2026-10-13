package mcp

import (
	"context"
	"encoding/json"
	"errors"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ihugang/pingcode-mcp-go/internal/auth"
	"github.com/ihugang/pingcode-mcp-go/internal/pingcode"
	"github.com/ihugang/pingcode-mcp-go/internal/security"
)

const (
	ErrorValidation        = "VALIDATION"
	ErrorAuthentication    = "AUTHENTICATION"
	ErrorPermission        = "PERMISSION"
	ErrorNotFound          = "NOT_FOUND"
	ErrorRateLimited       = "RATE_LIMITED"
	ErrorUpstream          = "UPSTREAM"
	ErrorTimeout           = "TIMEOUT"
	ErrorCancelled         = "CANCELLED"
	ErrorReadOnly          = "READ_ONLY"
	ErrorProjectNotAllowed = "PROJECT_NOT_ALLOWED"
)

// ErrorBody is the stable error envelope returned by a tool.
type ErrorBody struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ErrorPayload is returned both as JSON text and structured MCP content.
type ErrorPayload struct {
	Error ErrorBody `json:"error"`
}

// classifyError follows the public error-classification order from M6-1.
func classifyError(err error) ErrorPayload {
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorPayload{Error: ErrorBody{Type: ErrorTimeout, Message: "request timed out"}}
	}
	if errors.Is(err, context.Canceled) {
		return ErrorPayload{Error: ErrorBody{Type: ErrorCancelled, Message: "request canceled"}}
	}

	var policyErr *security.Error
	if errors.As(err, &policyErr) {
		typ := ErrorProjectNotAllowed
		if policyErr.Kind == security.KindReadOnly {
			typ = ErrorReadOnly
		}
		return ErrorPayload{Error: ErrorBody{Type: typ, Message: policyErr.Error()}}
	}

	var apiErr *pingcode.Error
	if errors.As(err, &apiErr) {
		typ := ErrorUpstream
		switch apiErr.Kind {
		case pingcode.KindValidation:
			typ = ErrorValidation
		case pingcode.KindAuthentication:
			typ = ErrorAuthentication
		case pingcode.KindPermission:
			typ = ErrorPermission
		case pingcode.KindNotFound:
			typ = ErrorNotFound
		case pingcode.KindRateLimited:
			typ = ErrorRateLimited
		case pingcode.KindUpstream:
			typ = ErrorUpstream
		}
		return ErrorPayload{Error: ErrorBody{Type: typ, Message: apiErr.Error()}}
	}

	var authErr *auth.Error
	if errors.As(err, &authErr) {
		typ := ErrorUpstream
		if authErr.Kind == auth.KindNotConfigured || authErr.Kind == auth.KindAuthentication {
			typ = ErrorAuthentication
		}
		return ErrorPayload{Error: ErrorBody{Type: typ, Message: authErr.Error()}}
	}

	return ErrorPayload{Error: ErrorBody{Type: ErrorUpstream, Message: "internal error"}}
}

func toolError(err error) (*sdk.CallToolResult, any, error) {
	return errorResult(classifyError(err))
}

func validationError(message string) (*sdk.CallToolResult, any, error) {
	return errorResult(ErrorPayload{Error: ErrorBody{Type: ErrorValidation, Message: message}})
}

func errorResult(payload ErrorPayload) (*sdk.CallToolResult, any, error) {
	encoded, _ := json.Marshal(payload)
	result := &sdk.CallToolResult{
		IsError: true,
		Content: []sdk.Content{&sdk.TextContent{Text: string(encoded)}},
	}
	return result, payload, nil
}
