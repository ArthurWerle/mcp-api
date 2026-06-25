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

	start := time.Now()
	c.logger.Info("upstream request", "method", "GET", "url", u)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		c.logger.Error("failed to build request", "url", u, "err", err)
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		c.logger.Error("upstream unreachable", "url", u, "duration_ms", time.Since(start).Milliseconds(), "err", err)
		return nil, fmt.Errorf("could not reach upstream %s: %w", u, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read response body", "url", u, "err", err)
		return nil, err
	}

	duration := time.Since(start).Milliseconds()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.logger.Error("upstream error response", "url", u, "status", resp.StatusCode, "duration_ms", duration, "body", string(body))
		return nil, fmt.Errorf("upstream returned %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("upstream response ok", "url", u, "status", resp.StatusCode, "duration_ms", duration)
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
