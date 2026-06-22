package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterTools(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	registerListTransactions(s, client, logger)
	registerGetTransaction(s, client, logger)
	registerGetLatestTransactions(s, client, logger)
	registerGetBiggestTransactions(s, client, logger)
	registerGetAverageByType(s, client, logger)
	registerGetAverageByCategory(s, client, logger)
	registerListCategories(s, client, logger)
	registerListSubcategories(s, client, logger)
	registerHealthCheck(s, client, logger)
}

func toolResult(body []byte, err error, logger *slog.Logger, toolName string) (*mcp.CallToolResult, error) {
	if err != nil {
		logger.Error(toolName+" failed", "err", err)
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(body)), nil
}

func registerListTransactions(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("list_transactions",
		mcp.WithDescription("List transactions with optional filters"),
		mcp.WithBoolean("current_month", mcp.Description("Filter to current month only")),
		mcp.WithString("category", mcp.Description("Filter by category name")),
		mcp.WithString("query", mcp.Description("Search query")),
		mcp.WithString("type", mcp.Description("Transaction type: income or expense")),
		mcp.WithString("start_date", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD)")),
		mcp.WithNumber("limit", mcp.Description("Number of results to return")),
		mcp.WithNumber("offset", mcp.Description("Offset for pagination")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if req.GetBool("current_month", false) {
			q.Set("current_month", "true")
		}
		for _, key := range []string{"category", "query", "type", "start_date", "end_date"} {
			if v := req.GetString(key, ""); v != "" {
				q.Set(key, v)
			}
		}
		if v := req.GetInt("limit", 0); v > 0 {
			q.Set("limit", fmt.Sprintf("%d", v))
		}
		if v := req.GetInt("offset", 0); v > 0 {
			q.Set("offset", fmt.Sprintf("%d", v))
		}
		body, err := client.ListTransactions(ctx, q)
		return toolResult(body, err, logger, "list_transactions")
	})
}

func registerGetTransaction(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("get_transaction",
		mcp.WithDescription("Get a single transaction by ID"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Transaction ID")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		body, err := client.GetTransaction(ctx, id)
		return toolResult(body, err, logger, "get_transaction")
	})
}

func registerGetLatestTransactions(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("get_latest_transactions",
		mcp.WithDescription("Get the most recent transactions"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body, err := client.GetLatestTransactions(ctx)
		return toolResult(body, err, logger, "get_latest_transactions")
	})
}

func registerGetBiggestTransactions(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("get_biggest_transactions",
		mcp.WithDescription("Get the biggest transactions for a given month and year"),
		mcp.WithNumber("month", mcp.Description("Month (1-12)")),
		mcp.WithNumber("year", mcp.Description("Year (e.g. 2025)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if v := req.GetInt("month", 0); v > 0 {
			q.Set("month", fmt.Sprintf("%d", v))
		}
		if v := req.GetInt("year", 0); v > 0 {
			q.Set("year", fmt.Sprintf("%d", v))
		}
		body, err := client.GetBiggestTransactions(ctx, q)
		return toolResult(body, err, logger, "get_biggest_transactions")
	})
}

func registerGetAverageByType(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("get_average_by_type",
		mcp.WithDescription("Get average transaction amount grouped by type (income/expense)"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body, err := client.GetAverageByType(ctx)
		return toolResult(body, err, logger, "get_average_by_type")
	})
}

func registerGetAverageByCategory(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("get_average_by_category",
		mcp.WithDescription("Get average transaction amount grouped by category"),
		mcp.WithString("start_date", mcp.Description("Start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("End date (YYYY-MM-DD)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		for _, key := range []string{"start_date", "end_date"} {
			if v := req.GetString(key, ""); v != "" {
				q.Set(key, v)
			}
		}
		body, err := client.GetAverageByCategory(ctx, q)
		return toolResult(body, err, logger, "get_average_by_category")
	})
}

func registerListCategories(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("list_categories",
		mcp.WithDescription("List all transaction categories"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body, err := client.ListCategories(ctx)
		return toolResult(body, err, logger, "list_categories")
	})
}

func registerListSubcategories(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("list_subcategories",
		mcp.WithDescription("List all transaction subcategories"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body, err := client.ListSubcategories(ctx)
		return toolResult(body, err, logger, "list_subcategories")
	})
}

func registerHealthCheck(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("health_check",
		mcp.WithDescription("Check the health of the transactions service"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body, err := client.HealthCheck(ctx)
		return toolResult(body, err, logger, "health_check")
	})
}
