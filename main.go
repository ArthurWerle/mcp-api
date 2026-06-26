package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

// defaultInstructions is sent to MCP clients in the initialize response. It is
// the closest equivalent to a "system prompt" for an MCP server: ambient context
// the model can use to understand the data and pick the right tools. Override it
// at runtime with the MCP_INSTRUCTIONS environment variable.
const defaultInstructions = `This server exposes a personal finance / transactions dataset.

Data model:
- A transaction has an amount, a type ("income" or "expense"), a category and an
  optional subcategory, and a date.
- Amounts are stored in the account's base currency. Expenses and income are
  distinguished by the "type" field, not by the sign of the amount.
- Categories and subcategories are user-defined; use list_categories and
  list_subcategories to discover valid values before filtering by them.

Recurring vs. non-recurring transactions:
- A normal (non-recurring) transaction has a single specific date and represents
  one event (e.g. a $100 purchase on a given day).
- A recurring transaction (is_recurring = true), such as a subscription, does NOT
  have a single date. Instead it has a start date and (sometimes) an end date,
  and it represents a charge that repeats every month within that range.
- For ALL calculations, treat a recurring transaction as if it occurs once in
  every month between its start date and end date (inclusive). If there is no end
  date, treat it as ongoing through the period being analyzed (e.g. up to today
  or the end of the requested range).
- Example: a $20/month subscription that started 3 months ago and has no end date
  contributes $20 to each of those months and continues going forward — it is not
  a single $20 charge.
- When summing, averaging, or reporting spend over a period, expand recurring
  transactions across the relevant months before aggregating, so monthly totals
  and averages reflect the repeated charges.

Tool guidance:
- Use list_transactions for general queries; it supports filters for date range,
  category, type, free-text query, current month, and pagination (limit/offset).
- Use get_latest_transactions for the most recent activity and
  get_biggest_transactions for the largest items in a specific month/year.
- Use get_average_by_type and get_average_by_category for aggregate analysis.
- Dates are formatted as YYYY-MM-DD. Prefer explicit start_date/end_date over
  guessing ranges, and confirm the current date with the user when relevant.

How to answer:
- Always ask additional questions when you are not sure about something. Do not
  guess at the user's intent, the time range, or how to treat ambiguous data —
  ask first.
- Dedicate a significant amount of effort to gathering enough information before
  answering. Call the relevant tools, inspect the actual data (including how
  recurring transactions fall within the period), and verify your assumptions
  before giving a final answer.`

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func healthHandler(client *TransactionClient, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type response struct {
			Status     string `json:"status"`
			Service    string `json:"service"`
			Backend    string `json:"backend"`
			BackendURL string `json:"backend_url"`
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
	port := getEnv("SERVER_PORT", "3006")

	logger.Info("starting mcp-api",
		"transport", transport,
		"transaction_service_url", baseURL,
	)

	client := NewTransactionClient(baseURL, logger)

	instructions := getEnv("MCP_INSTRUCTIONS", defaultInstructions)

	s := server.NewMCPServer(
		"mcp-api",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithInstructions(instructions),
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
		sseServer := server.NewSSEServer(s, server.WithBaseURL(fmt.Sprintf("http://0.0.0.0:%s", port)))

		mux := http.NewServeMux()
		mux.HandleFunc("/health", healthHandler(client, logger))
		mux.Handle("/", sseServer)

		logger.Info("starting MCP server in HTTP/SSE mode", "addr", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	default:
		logger.Error("unknown TRANSPORT", "transport", transport)
		os.Exit(1)
	}
}
