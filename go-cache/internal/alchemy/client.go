package alchemy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"nft-proxy/internal/metrics"

	"github.com/status-im/proxy-common/httpclient"
)

// Client represents an Alchemy NFT API client
type Client struct {
	apiKey     string
	baseURLs   map[string]string
	httpClient *httpclient.HTTPClientWithRetries
}

// NewClient creates a new Alchemy API client
func NewClient(apiKey string, baseURLs map[string]string, retryOpts httpclient.RetryOptions) *Client {
	statusHandler := metrics.NewAlchemyHTTPMetrics()
	return &Client{
		apiKey:     apiKey,
		baseURLs:   baseURLs,
		httpClient: httpclient.NewHTTPClientWithRetries(retryOpts, statusHandler, nil),
	}
}

// getBaseURL returns the base URL for a given chain and network
func (c *Client) getBaseURL(chain, network string) (string, error) {
	chainKey := strings.ToLower(chain) + "-" + strings.ToLower(network)
	baseURL, ok := c.baseURLs[chainKey]
	if !ok {
		return "", fmt.Errorf("unsupported chain: %s-%s", chain, network)
	}
	return baseURL, nil
}

// ProxyGET forwards a GET request to Alchemy and returns the raw response
func (c *Client) ProxyGET(ctx context.Context, chain, network, path, rawQuery string) ([]byte, int, error) {
	baseURL, err := c.getBaseURL(chain, network)
	if err != nil {
		return nil, 0, err
	}

	// Build full URL with API key in path (format: /nft/v3/{apiKey}/{path})
	endpoint := fmt.Sprintf("%s/nft/v3/%s%s", baseURL, c.apiKey, path)
	if rawQuery != "" {
		endpoint = fmt.Sprintf("%s?%s", endpoint, rawQuery)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, respBody, _, err := c.httpClient.ExecuteRequest(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return respBody, resp.StatusCode, nil
}

// ProxyPOST forwards a POST request to Alchemy and returns the raw response
func (c *Client) ProxyPOST(ctx context.Context, chain, network, path string, body []byte) ([]byte, int, error) {
	baseURL, err := c.getBaseURL(chain, network)
	if err != nil {
		return nil, 0, err
	}

	// Build full URL with API key in path (format: /nft/v3/{apiKey}/{path})
	endpoint := fmt.Sprintf("%s/nft/v3/%s%s", baseURL, c.apiKey, path)

	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = bytes.NewBuffer(body)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, respBody, _, err := c.httpClient.ExecuteRequest(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return respBody, resp.StatusCode, nil
}
