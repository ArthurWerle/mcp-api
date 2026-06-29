package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// capturedRequest records what the MCP server sent to the backend.
type capturedRequest struct {
	method string
	path   string
	body   map[string]any
}

// newTestHarness spins up a mock backend that records the last request, wires a
// fully-registered MCP server in front of it, and returns an initialized
// in-process client plus a pointer to the captured request.
func newTestHarness(t *testing.T) (*client.Client, *capturedRequest) {
	t.Helper()

	var mu sync.Mutex
	captured := &capturedRequest{}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		mu.Lock()
		captured.method = r.Method
		captured.path = r.URL.Path
		captured.body = nil
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &captured.body)
		}
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(backend.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tc := NewTransactionClient(backend.URL, logger)

	s := server.NewMCPServer("mcp-api-test", "test", server.WithToolCapabilities(false))
	RegisterTools(s, tc, logger)

	c, err := client.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("client start: %v", err)
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "test", Version: "1.0.0"}
	if _, err := c.Initialize(context.Background(), initReq); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	return c, captured
}

func callTool(t *testing.T, c *client.Client, name string, args map[string]any) {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	res, err := c.CallTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("tool %s returned error: %+v", name, res.Content)
	}
}

func TestCreateTransactionSendsCorrectRequest(t *testing.T) {
	c, captured := newTestHarness(t)

	callTool(t, c, "create_transaction", map[string]any{
		"amount":      10.5,
		"type":        "expense",
		"category_id": 3,
		"location":    "Padaria",
		"description": "lunch",
	})

	if captured.method != http.MethodPost || captured.path != "/transactions" {
		t.Fatalf("expected POST /transactions, got %s %s", captured.method, captured.path)
	}
	if captured.body["amount"] != 10.5 {
		t.Errorf("amount = %v, want 10.5", captured.body["amount"])
	}
	if captured.body["type"] != "expense" {
		t.Errorf("type = %v, want expense", captured.body["type"])
	}
	// created_by_id must default to 1 when not supplied.
	if captured.body["created_by_id"] != float64(1) {
		t.Errorf("created_by_id = %v, want 1", captured.body["created_by_id"])
	}
	if captured.body["category_id"] != float64(3) {
		t.Errorf("category_id = %v, want 3", captured.body["category_id"])
	}
	if captured.body["location"] != "Padaria" {
		t.Errorf("location = %v, want Padaria", captured.body["location"])
	}
}

func TestCreateTransactionValidatesType(t *testing.T) {
	c, _ := newTestHarness(t)

	req := mcp.CallToolRequest{}
	req.Params.Name = "create_transaction"
	req.Params.Arguments = map[string]any{"amount": 10.0, "type": "bogus"}
	res, err := c.CallTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected validation error for invalid type, got success")
	}
}

func TestUpdateTransactionIsPartial(t *testing.T) {
	c, captured := newTestHarness(t)

	callTool(t, c, "update_transaction", map[string]any{
		"id":          "42",
		"description": "updated",
	})

	if captured.method != http.MethodPut || captured.path != "/transactions/42" {
		t.Fatalf("expected PUT /transactions/42, got %s %s", captured.method, captured.path)
	}
	// Only the provided field should be sent; no created_by_id, amount, etc.
	if len(captured.body) != 1 {
		t.Fatalf("expected exactly 1 field in body, got %v", captured.body)
	}
	if captured.body["description"] != "updated" {
		t.Errorf("description = %v, want updated", captured.body["description"])
	}
}

func TestCreateCategoryAndSubcategory(t *testing.T) {
	c, captured := newTestHarness(t)

	callTool(t, c, "create_category", map[string]any{"name": "Food", "color": "#fff"})
	if captured.method != http.MethodPost || captured.path != "/categories" {
		t.Fatalf("expected POST /categories, got %s %s", captured.method, captured.path)
	}
	if captured.body["name"] != "Food" || captured.body["color"] != "#fff" {
		t.Errorf("unexpected category body: %v", captured.body)
	}

	callTool(t, c, "create_subcategory", map[string]any{"name": "Vegetables"})
	if captured.method != http.MethodPost || captured.path != "/subcategories" {
		t.Fatalf("expected POST /subcategories, got %s %s", captured.method, captured.path)
	}
	if captured.body["name"] != "Vegetables" {
		t.Errorf("unexpected subcategory body: %v", captured.body)
	}
}

func TestLocationToolsAndList(t *testing.T) {
	c, captured := newTestHarness(t)

	callTool(t, c, "create_location", map[string]any{"name": "Mercado X"})
	if captured.method != http.MethodPost || captured.path != "/locations" {
		t.Fatalf("expected POST /locations, got %s %s", captured.method, captured.path)
	}
	if captured.body["name"] != "Mercado X" {
		t.Errorf("unexpected location body: %v", captured.body)
	}

	callTool(t, c, "update_location", map[string]any{"id": "7", "name": "Mercado Y"})
	if captured.method != http.MethodPut || captured.path != "/locations/7" {
		t.Fatalf("expected PUT /locations/7, got %s %s", captured.method, captured.path)
	}

	callTool(t, c, "list_locations", map[string]any{})
	if captured.method != http.MethodGet || captured.path != "/locations" {
		t.Fatalf("expected GET /locations, got %s %s", captured.method, captured.path)
	}
}

func TestWriteToolsAreNotReadOnly(t *testing.T) {
	c, _ := newTestHarness(t)

	res, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	writeTools := map[string]bool{
		"create_transaction": true, "update_transaction": true,
		"create_category": true, "update_category": true,
		"create_subcategory": true, "update_subcategory": true,
		"create_location": true, "update_location": true,
	}
	seen := map[string]bool{}
	for _, tool := range res.Tools {
		seen[tool.Name] = true
		if writeTools[tool.Name] {
			if tool.Annotations.ReadOnlyHint == nil || *tool.Annotations.ReadOnlyHint {
				t.Errorf("write tool %s should have ReadOnlyHint=false", tool.Name)
			}
		}
	}
	for name := range writeTools {
		if !seen[name] {
			t.Errorf("expected tool %s to be registered", name)
		}
	}
}
