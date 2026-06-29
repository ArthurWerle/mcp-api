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
	registerListLocations(s, client, logger)
	registerCreateTransaction(s, client, logger)
	registerUpdateTransaction(s, client, logger)
	registerCreateCategory(s, client, logger)
	registerUpdateCategory(s, client, logger)
	registerCreateSubcategory(s, client, logger)
	registerUpdateSubcategory(s, client, logger)
	registerCreateLocation(s, client, logger)
	registerUpdateLocation(s, client, logger)
	registerHealthCheck(s, client, logger)
}

// defaultCreatedByID is the owner attached to transactions created through the
// MCP server. The backend marks created_by_id as NOT NULL but does not validate
// it, so we always send an explicit value instead of letting it default to 0.
const defaultCreatedByID = 1

// pickArgs copies the given keys from the request arguments into a payload map,
// but only when the caller actually provided them. Values are passed through
// untouched so JSON types (numbers, booleans, strings) reach the backend as-is.
// This preserves partial-update semantics: omitted fields are left out of the
// body, so the backend keeps their current values.
func pickArgs(req mcp.CallToolRequest, keys ...string) map[string]any {
	args := req.GetArguments()
	payload := make(map[string]any)
	for _, key := range keys {
		if v, ok := args[key]; ok {
			payload[key] = v
		}
	}
	return payload
}

func toolResult(body []byte, err error, logger *slog.Logger, toolName string) (*mcp.CallToolResult, error) {
	if err != nil {
		logger.Error("tool call failed", "tool", toolName, "err", err)
		return mcp.NewToolResultError(err.Error()), nil
	}
	logger.Info("tool call succeeded", "tool", toolName)
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
		logger.Info("tool call", "tool", "list_transactions", "params", q.Encode())
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
		logger.Info("tool call", "tool", "get_transaction", "id", id)
		body, err := client.GetTransaction(ctx, id)
		return toolResult(body, err, logger, "get_transaction")
	})
}

func registerGetLatestTransactions(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("get_latest_transactions",
		mcp.WithDescription("Get the most recent transactions"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call", "tool", "get_latest_transactions")
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
		logger.Info("tool call", "tool", "get_biggest_transactions", "params", q.Encode())
		body, err := client.GetBiggestTransactions(ctx, q)
		return toolResult(body, err, logger, "get_biggest_transactions")
	})
}

func registerGetAverageByType(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("get_average_by_type",
		mcp.WithDescription("Get average transaction amount grouped by type (income/expense)"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call", "tool", "get_average_by_type")
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
		logger.Info("tool call", "tool", "get_average_by_category", "params", q.Encode())
		body, err := client.GetAverageByCategory(ctx, q)
		return toolResult(body, err, logger, "get_average_by_category")
	})
}

func registerListCategories(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("list_categories",
		mcp.WithDescription("List all transaction categories"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call", "tool", "list_categories")
		body, err := client.ListCategories(ctx)
		return toolResult(body, err, logger, "list_categories")
	})
}

func registerListSubcategories(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("list_subcategories",
		mcp.WithDescription("List all transaction subcategories"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call", "tool", "list_subcategories")
		body, err := client.ListSubcategories(ctx)
		return toolResult(body, err, logger, "list_subcategories")
	})
}

func registerListLocations(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("list_locations",
		mcp.WithDescription("List all locations (places/merchants attached to transactions)"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call", "tool", "list_locations")
		body, err := client.ListLocations(ctx)
		return toolResult(body, err, logger, "list_locations")
	})
}

func registerCreateTransaction(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("create_transaction",
		mcp.WithDescription("Create a new transaction. amount and type are required. "+
			"Reference categories/subcategories by id (discover them with list_categories / "+
			"list_subcategories); 'location' is free text and is matched/created automatically."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithNumber("amount", mcp.Required(), mcp.Description("Transaction amount (must be greater than 0)")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Transaction type: income or expense")),
		mcp.WithNumber("category_id", mcp.Description("Category ID (see list_categories)")),
		mcp.WithNumber("subcategory_id", mcp.Description("Subcategory ID (see list_subcategories)")),
		mcp.WithNumber("created_by_id", mcp.Description("Owner user ID (defaults to 1)")),
		mcp.WithString("description", mcp.Description("Free-text description")),
		mcp.WithString("subtype", mcp.Description("Optional subtype: salary, profits, or pro-labore")),
		mcp.WithBoolean("is_recurring", mcp.Description("Whether this is a recurring transaction")),
		mcp.WithString("frequency", mcp.Description("Recurrence frequency, e.g. monthly")),
		mcp.WithString("date", mcp.Description("Transaction date in RFC3339 (e.g. 2024-01-15T10:30:00Z)")),
		mcp.WithString("start_date", mcp.Description("Recurring start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("Recurring end date (YYYY-MM-DD)")),
		mcp.WithString("location", mcp.Description("Place/merchant name (free text, deduplicated automatically)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		amount, err := req.RequireFloat("amount")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if amount <= 0 {
			return mcp.NewToolResultError("amount must be greater than 0"), nil
		}
		txType, err := req.RequireString("type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if txType != "income" && txType != "expense" {
			return mcp.NewToolResultError("type must be either 'income' or 'expense'"), nil
		}

		payload := pickArgs(req, "category_id", "subcategory_id", "description", "subtype",
			"is_recurring", "frequency", "date", "start_date", "end_date", "location")
		payload["amount"] = amount
		payload["type"] = txType
		payload["created_by_id"] = req.GetInt("created_by_id", defaultCreatedByID)

		logger.Info("tool call", "tool", "create_transaction", "amount", amount, "type", txType)
		body, err := client.CreateTransaction(ctx, payload)
		return toolResult(body, err, logger, "create_transaction")
	})
}

func registerUpdateTransaction(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("update_transaction",
		mcp.WithDescription("Update an existing transaction by ID. Only the fields you provide are changed."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("id", mcp.Required(), mcp.Description("Transaction ID")),
		mcp.WithNumber("amount", mcp.Description("Transaction amount (must be greater than 0)")),
		mcp.WithString("type", mcp.Description("Transaction type: income or expense")),
		mcp.WithNumber("category_id", mcp.Description("Category ID (see list_categories)")),
		mcp.WithNumber("subcategory_id", mcp.Description("Subcategory ID (see list_subcategories)")),
		mcp.WithString("description", mcp.Description("Free-text description")),
		mcp.WithString("subtype", mcp.Description("Optional subtype: salary, profits, or pro-labore")),
		mcp.WithBoolean("is_recurring", mcp.Description("Whether this is a recurring transaction")),
		mcp.WithString("frequency", mcp.Description("Recurrence frequency, e.g. monthly")),
		mcp.WithString("date", mcp.Description("Transaction date in RFC3339 (e.g. 2024-01-15T10:30:00Z)")),
		mcp.WithString("start_date", mcp.Description("Recurring start date (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("Recurring end date (YYYY-MM-DD)")),
		mcp.WithString("location", mcp.Description("Place/merchant name (free text, deduplicated automatically)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if v := req.GetFloat("amount", 0); v != 0 && v <= 0 {
			return mcp.NewToolResultError("amount must be greater than 0"), nil
		}
		if v := req.GetString("type", ""); v != "" && v != "income" && v != "expense" {
			return mcp.NewToolResultError("type must be either 'income' or 'expense'"), nil
		}

		payload := pickArgs(req, "amount", "type", "category_id", "subcategory_id", "description",
			"subtype", "is_recurring", "frequency", "date", "start_date", "end_date", "location")

		logger.Info("tool call", "tool", "update_transaction", "id", id)
		body, err := client.UpdateTransaction(ctx, id, payload)
		return toolResult(body, err, logger, "update_transaction")
	})
}

func registerCreateCategory(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("create_category",
		mcp.WithDescription("Create a new category. name is required."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("name", mcp.Required(), mcp.Description("Category name")),
		mcp.WithString("description", mcp.Description("Optional description")),
		mcp.WithString("color", mcp.Description("Optional color (e.g. hex code like #FF5733)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		payload := pickArgs(req, "description", "color")
		payload["name"] = name

		logger.Info("tool call", "tool", "create_category", "name", name)
		body, err := client.CreateCategory(ctx, payload)
		return toolResult(body, err, logger, "create_category")
	})
}

func registerUpdateCategory(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("update_category",
		mcp.WithDescription("Update an existing category by ID. Only the fields you provide are changed."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("id", mcp.Required(), mcp.Description("Category ID")),
		mcp.WithString("name", mcp.Description("Category name")),
		mcp.WithString("description", mcp.Description("Optional description")),
		mcp.WithString("color", mcp.Description("Optional color (e.g. hex code like #FF5733)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		payload := pickArgs(req, "name", "description", "color")

		logger.Info("tool call", "tool", "update_category", "id", id)
		body, err := client.UpdateCategory(ctx, id, payload)
		return toolResult(body, err, logger, "update_category")
	})
}

func registerCreateSubcategory(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("create_subcategory",
		mcp.WithDescription("Create a new subcategory. name is required."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("name", mcp.Required(), mcp.Description("Subcategory name")),
		mcp.WithString("description", mcp.Description("Optional description")),
		mcp.WithString("color", mcp.Description("Optional color (e.g. hex code like #00FF00)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		payload := pickArgs(req, "description", "color")
		payload["name"] = name

		logger.Info("tool call", "tool", "create_subcategory", "name", name)
		body, err := client.CreateSubcategory(ctx, payload)
		return toolResult(body, err, logger, "create_subcategory")
	})
}

func registerUpdateSubcategory(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("update_subcategory",
		mcp.WithDescription("Update an existing subcategory by ID. Only the fields you provide are changed."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("id", mcp.Required(), mcp.Description("Subcategory ID")),
		mcp.WithString("name", mcp.Description("Subcategory name")),
		mcp.WithString("description", mcp.Description("Optional description")),
		mcp.WithString("color", mcp.Description("Optional color (e.g. hex code like #00FF00)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		payload := pickArgs(req, "name", "description", "color")

		logger.Info("tool call", "tool", "update_subcategory", "id", id)
		body, err := client.UpdateSubcategory(ctx, id, payload)
		return toolResult(body, err, logger, "update_subcategory")
	})
}

func registerCreateLocation(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("create_location",
		mcp.WithDescription("Create a location by name. If a location with the same normalized "+
			"name already exists, the existing one is returned (no duplicate is created)."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("name", mcp.Required(), mcp.Description("Location name")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		logger.Info("tool call", "tool", "create_location", "name", name)
		body, err := client.CreateLocation(ctx, map[string]any{"name": name})
		return toolResult(body, err, logger, "create_location")
	})
}

func registerUpdateLocation(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("update_location",
		mcp.WithDescription("Update an existing location's name by ID."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("id", mcp.Required(), mcp.Description("Location ID")),
		mcp.WithString("name", mcp.Description("New location name")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		payload := pickArgs(req, "name")

		logger.Info("tool call", "tool", "update_location", "id", id)
		body, err := client.UpdateLocation(ctx, id, payload)
		return toolResult(body, err, logger, "update_location")
	})
}

func registerHealthCheck(s *server.MCPServer, client *TransactionClient, logger *slog.Logger) {
	tool := mcp.NewTool("health_check",
		mcp.WithDescription("Check the health of the transactions service"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call", "tool", "health_check")
		body, err := client.HealthCheck(ctx)
		return toolResult(body, err, logger, "health_check")
	})
}
