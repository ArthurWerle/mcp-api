package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
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

// send performs a request with a JSON body (POST/PUT) and surfaces non-2xx
// responses as errors, mirroring get's logging and error handling.
func (c *TransactionClient) send(ctx context.Context, method, path string, payload any) ([]byte, error) {
	u := c.baseURL + path

	encoded, err := json.Marshal(payload)
	if err != nil {
		c.logger.Error("failed to marshal request body", "url", u, "err", err)
		return nil, err
	}

	start := time.Now()
	c.logger.Info("upstream request", "method", method, "url", u)

	req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(encoded))
	if err != nil {
		c.logger.Error("failed to build request", "url", u, "err", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

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

func (c *TransactionClient) post(ctx context.Context, path string, payload any) ([]byte, error) {
	return c.send(ctx, http.MethodPost, path, payload)
}

func (c *TransactionClient) put(ctx context.Context, path string, payload any) ([]byte, error) {
	return c.send(ctx, http.MethodPut, path, payload)
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

func (c *TransactionClient) ListLocations(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "/locations", nil)
}

func (c *TransactionClient) CreateTransaction(ctx context.Context, payload any) ([]byte, error) {
	return c.post(ctx, "/transactions", payload)
}

func (c *TransactionClient) UpdateTransaction(ctx context.Context, id string, payload any) ([]byte, error) {
	return c.put(ctx, "/transactions/"+id, payload)
}

func (c *TransactionClient) CreateCategory(ctx context.Context, payload any) ([]byte, error) {
	return c.post(ctx, "/categories", payload)
}

func (c *TransactionClient) UpdateCategory(ctx context.Context, id string, payload any) ([]byte, error) {
	return c.put(ctx, "/categories/"+id, payload)
}

func (c *TransactionClient) CreateSubcategory(ctx context.Context, payload any) ([]byte, error) {
	return c.post(ctx, "/subcategories", payload)
}

func (c *TransactionClient) UpdateSubcategory(ctx context.Context, id string, payload any) ([]byte, error) {
	return c.put(ctx, "/subcategories/"+id, payload)
}

func (c *TransactionClient) CreateLocation(ctx context.Context, payload any) ([]byte, error) {
	return c.post(ctx, "/locations", payload)
}

func (c *TransactionClient) UpdateLocation(ctx context.Context, id string, payload any) ([]byte, error) {
	return c.put(ctx, "/locations/"+id, payload)
}

func (c *TransactionClient) HealthCheck(ctx context.Context) ([]byte, error) {
	// /health lives at the root, not under /api/v2
	u := strings.TrimSuffix(c.baseURL, "/api/v2") + "/health"
	return c.getURL(ctx, u)
}

func (c *TransactionClient) getURL(ctx context.Context, u string) ([]byte, error) {
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
