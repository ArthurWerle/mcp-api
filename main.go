package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	logLevel := slog.LevelInfo
	if getEnv("LOG_LEVEL", "info") == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	baseURL := getEnv("TRANSACTION_SERVICE_URL", "http://localhost:1235/api/v2")
	transport := getEnv("TRANSPORT", "http")
	port := getEnv("SERVER_PORT", "3001")
	publicURL := getEnv("PUBLIC_URL", fmt.Sprintf("http://0.0.0.0:%s", port))

	client := NewTransactionClient(baseURL, logger)

	s := server.NewMCPServer(
		"mcp-api",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	RegisterTools(s, client, logger)

	switch transport {
	case "stdio":
		logger.Info("starting MCP server in stdio mode")
		if err := server.ServeStdio(s); err != nil {
			logger.Error("stdio server error", "err", err)
			os.Exit(1)
		}
	case "http":
		addr := fmt.Sprintf(":%s", port)
		logger.Info("starting MCP server in HTTP/SSE mode", "addr", addr)
		sseServer := server.NewSSEServer(s, server.WithBaseURL(publicURL))
		if err := sseServer.Start(addr); err != nil {
			logger.Error("SSE server error", "err", err)
			os.Exit(1)
		}
	default:
		logger.Error("unknown TRANSPORT", "transport", transport)
		os.Exit(1)
	}

}
