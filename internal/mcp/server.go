package mcp

import (
	"log/slog"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ihugang/pingcode-mcp-go/internal/security"
)

// Options configures the MCP server identity and SDK logger.
type Options struct {
	Version string
	Logger  *slog.Logger
}

// NewServer registers the six supported P0 tools. A nil API or policy is a
// composition-root programming error and panics; callers must provide both.
func NewServer(api PingCodeAPI, policy *security.Policy, opts Options) *sdk.Server {
	if api == nil {
		panic("mcp: PingCode API is nil")
	}
	if policy == nil {
		panic("mcp: security policy is nil")
	}

	server := sdk.NewServer(&sdk.Implementation{Name: "pingcode-mcp-go", Version: opts.Version}, &sdk.ServerOptions{Logger: opts.Logger})
	registerTools(server, api, policy)
	return server
}
