package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	cfg        config
	httpClient *http.Client
}

func New(opts ...Option) *Client {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.timeout,
		},
	}
}

func (c *Client) Get(ctx context.Context, path string, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodGet, path, nil, opts...)
}

func (c *Client) Post(ctx context.Context, path string, body any, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodPost, path, body, opts...)
}

func (c *Client) Put(ctx context.Context, path string, body any, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodPut, path, body, opts...)
}

func (c *Client) Patch(ctx context.Context, path string, body any, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodPatch, path, body, opts...)
}

func (c *Client) Delete(ctx context.Context, path string, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodDelete, path, nil, opts...)
}

func (c *Client) do(ctx context.Context, method, path string, body any, opts ...RequestOption) (*Response, error) {
	reqCfg := defaultRequestConfig()
	for _, opt := range opts {
		opt(&reqCfg)
	}

	fullURL := c.buildURL(path, reqCfg.queryParams)

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply global headers
	for k, v := range c.cfg.headers {
		req.Header.Set(k, v)
	}

	// Apply request-specific headers (override global)
	for k, v := range reqCfg.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		raw:        rawBody,
	}, nil
}

func (c *Client) buildURL(path string, queryParams map[string]string) string {
	base := c.cfg.baseURL + path

	if len(queryParams) == 0 {
		return base
	}

	u, err := url.Parse(base)
	if err != nil {
		return base
	}

	q := u.Query()
	for k, v := range queryParams {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return u.String()
}
