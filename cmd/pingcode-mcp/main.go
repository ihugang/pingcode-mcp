// Command pingcode-mcp is the stdio composition root for OpenSpec change
// bootstrap-pingcode-mcp task 4.1 and PRD FR-003. PingCode demand ID: PENDING.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ihugang/pingcode-mcp-go/internal/auth"
	"github.com/ihugang/pingcode-mcp-go/internal/config"
	internalmcp "github.com/ihugang/pingcode-mcp-go/internal/mcp"
	"github.com/ihugang/pingcode-mcp-go/internal/pingcode"
	"github.com/ihugang/pingcode-mcp-go/internal/security"
)

const version = "0.1.0-dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}
	provider := auth.NewClientCredentialsProvider(auth.Credentials{
		ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, BaseURL: cfg.BaseURL,
	}, httpClient)
	policy := security.NewPolicy(cfg.ReadOnly, cfg.AllowedProjectIDs)
	client := pingcode.New(provider, policy, httpClient, cfg.BaseURL)
	server := internalmcp.NewServer(client, policy, internalmcp.Options{Version: version, Logger: logger})
	return server.Run(ctx, &sdk.StdioTransport{})
}
