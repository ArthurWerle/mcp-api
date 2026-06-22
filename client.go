package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type TransactionClient struct {
	baseURL string
	http    *http.Client
	logger  *slog.Logger
}

func NewTransactionClient(baseURL string, logger *slog.Logger) *TransactionClient {
	return &TransactionClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
		logger:  logger,
	}
}

func (c *TransactionClient) get(ctx context.Context, path string, q url.Values) ([]byte, error) {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	c.logger.Debug("GET", "url", u)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream returned %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *TransactionClient) ListTransactions(ctx context.Context, q url.Values) ([]byte, error) {
	return c.get(ctx, "/transactions", q)
}

func (c *TransactionClient) GetTransaction(ctx context.Context, id string) ([]byte, error) {
	return c.get(ctx, "/transactions/"+id, nil)
}

func (c *TransactionClient) GetLatestTransactions(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "/transactions/latest", nil)
}

func (c *TransactionClient) GetBiggestTransactions(ctx context.Context, q url.Values) ([]byte, error) {
	return c.get(ctx, "/transactions/biggest", q)
}

func (c *TransactionClient) GetAverageByType(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "/transactions/average/by-type", nil)
}

func (c *TransactionClient) GetAverageByCategory(ctx context.Context, q url.Values) ([]byte, error) {
	return c.get(ctx, "/transactions/average/by-category", q)
}

func (c *TransactionClient) ListCategories(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "/categories", nil)
}

func (c *TransactionClient) ListSubcategories(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "/subcategories", nil)
}

func (c *TransactionClient) HealthCheck(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "/health", nil)
}
