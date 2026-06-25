package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func startHealthServer(port string, client *TransactionClient, logger *slog.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		type response struct {
			Status      string `json:"status"`
			Service     string `json:"service"`
			Backend     string `json:"backend"`
			BackendURL  string `json:"backend_url"`
		}

		backendStatus := "ok"
		_, err := client.HealthCheck(r.Context())
		if err != nil {
			backendStatus = "unreachable: " + err.Error()
		}

		overall := "ok"
		if err != nil {
			overall = "degraded"
		}

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(response{
			Status:     overall,
			Service:    "mcp-api",
			Backend:    backendStatus,
			BackendURL: client.baseURL,
		})
	})

	addr := ":" + port
	logger.Info("starting health server", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("health server error", "err", err)
	}
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
	healthPort := getEnv("HEALTH_PORT", "3002")

	logger.Info("starting mcp-api",
		"transport", transport,
		"transaction_service_url", baseURL,
	)

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
		go startHealthServer(healthPort, client, logger)

		addr := fmt.Sprintf(":%s", port)
		logger.Info("starting MCP server in HTTP/SSE mode", "addr", addr)
		sseServer := server.NewSSEServer(s, server.WithBaseURL(fmt.Sprintf("http://0.0.0.0:%s", port)))
		if err := sseServer.Start(addr); err != nil {
			logger.Error("SSE server error", "err", err)
			os.Exit(1)
		}
	default:
		logger.Error("unknown TRANSPORT", "transport", transport)
		os.Exit(1)
	}

	// suppress unused import warning when health check ctx is used
	_ = context.Background
}
