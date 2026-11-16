package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/status-im/proxy-common/httpclient"
	"go.uber.org/zap"

	"nft-proxy/internal/alchemy"
)

func TestExtractAlchemyPath(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "getNFTsForOwner",
			url:      "/eth/mainnet/nft/v3/getNFTsForOwner?owner=0x123",
			expected: "/getNFTsForOwner",
		},
		{
			name:     "getOwnersForContract",
			url:      "/eth/mainnet/nft/v3/getOwnersForContract?contractAddress=0x456",
			expected: "/getOwnersForContract",
		},
		{
			name:     "getNFTMetadataBatch",
			url:      "/polygon/mainnet/nft/v3/getNFTMetadataBatch",
			expected: "/getNFTMetadataBatch",
		},
		{
			name:     "getContractMetadataBatch",
			url:      "/base/mainnet/nft/v3/getContractMetadataBatch",
			expected: "/getContractMetadataBatch",
		},
		{
			name:     "invalid path - no nft prefix",
			url:      "/eth/mainnet/some/other/path",
			expected: "",
		},
		{
			name:     "path with multiple segments",
			url:      "/eth/mainnet/nft/v3/some/nested/endpoint",
			expected: "/some/nested/endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			result := ExtractAlchemyPath(req)
			if result != tt.expected {
				t.Errorf("ExtractAlchemyPath() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHandleProxy_GET(t *testing.T) {
	mockAlchemy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ownedNfts": []map[string]string{
				{"name": "NFT 1"},
				{"name": "NFT 2"},
			},
		})
	}))
	defer mockAlchemy.Close()

	baseURLs := map[string]string{
		"eth-mainnet": mockAlchemy.URL,
	}
	client := alchemy.NewClient("test-api-key", baseURLs, httpclient.DefaultRetryOptions())
	logger := zap.NewNop()
	server := NewServer(client, logger)

	req := httptest.NewRequest(http.MethodGet, "/eth/mainnet/nft/v3/getNFTsForOwner?owner=0x123", nil)
	req = mux.SetURLVars(req, map[string]string{
		"chain":   "eth",
		"network": "mainnet",
	})
	w := httptest.NewRecorder()

	server.handleProxy(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["ownedNfts"]; !ok {
		t.Error("Expected ownedNfts field in response")
	}
}

func TestHandleProxy_POST(t *testing.T) {
	mockAlchemy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		var requestBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if _, ok := requestBody["tokens"]; !ok {
			t.Error("Expected tokens field in request body")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"nfts": []map[string]string{
				{"name": "NFT 1"},
			},
		})
	}))
	defer mockAlchemy.Close()

	baseURLs := map[string]string{
		"polygon-mainnet": mockAlchemy.URL,
	}
	client := alchemy.NewClient("test-api-key", baseURLs, httpclient.DefaultRetryOptions())
	logger := zap.NewNop()
	server := NewServer(client, logger)

	requestBody := map[string]interface{}{
		"tokens": []map[string]string{
			{"contractAddress": "0x123", "tokenId": "1"},
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/polygon/mainnet/nft/v3/getNFTMetadataBatch", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{
		"chain":   "polygon",
		"network": "mainnet",
	})
	w := httptest.NewRecorder()

	server.handleProxy(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["nfts"]; !ok {
		t.Error("Expected nfts field in response")
	}
}

func TestHandleProxy_InvalidPath(t *testing.T) {
	logger := zap.NewNop()
	client := alchemy.NewClient("test-api-key", map[string]string{}, httpclient.DefaultRetryOptions())
	server := NewServer(client, logger)

	req := httptest.NewRequest(http.MethodGet, "/eth/mainnet/invalid/path", nil)
	req = mux.SetURLVars(req, map[string]string{
		"chain":   "eth",
		"network": "mainnet",
	})
	w := httptest.NewRecorder()

	server.handleProxy(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Invalid request path" {
		t.Errorf("Expected error message 'Invalid request path', got %q", response["error"])
	}
}

func TestHandleProxy_MethodNotAllowed(t *testing.T) {
	logger := zap.NewNop()
	client := alchemy.NewClient("test-api-key", map[string]string{}, httpclient.DefaultRetryOptions())
	server := NewServer(client, logger)

	req := httptest.NewRequest(http.MethodPut, "/eth/mainnet/nft/v3/getNFTsForOwner", nil)
	req = mux.SetURLVars(req, map[string]string{
		"chain":   "eth",
		"network": "mainnet",
	})
	w := httptest.NewRecorder()

	server.handleProxy(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "Method not allowed" {
		t.Errorf("Expected error message 'Method not allowed', got %q", response["error"])
	}
}

func TestHandleProxy_UnsupportedChain(t *testing.T) {
	baseURLs := map[string]string{
		"eth-mainnet": "http://localhost:8080",
	}
	client := alchemy.NewClient("test-api-key", baseURLs, httpclient.DefaultRetryOptions())
	logger := zap.NewNop()
	server := NewServer(client, logger)

	req := httptest.NewRequest(http.MethodGet, "/unsupported/mainnet/nft/v3/getNFTsForOwner?owner=0x123", nil)
	req = mux.SetURLVars(req, map[string]string{
		"chain":   "unsupported",
		"network": "mainnet",
	})
	w := httptest.NewRecorder()

	server.handleProxy(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "unsupported chain: unsupported-mainnet" {
		t.Errorf("Expected error message 'unsupported chain: unsupported-mainnet', got %q", response["error"])
	}
}

// What Alchemy answers decides what the caller gets: its refusals are passed
// on, its failures and the proxy's own credentials trouble are a 502.
func TestHandleProxy_AlchemyErrors(t *testing.T) {
	tests := []struct {
		name        string
		alchemyCode int
		alchemyBody string
		wantCode    int
		wantBody    string
	}{
		{"BadRequestPassedOn", http.StatusBadRequest, `{"error":{"message":"owner is required"}}`, http.StatusBadRequest, `{"error":{"message":"owner is required"}}`},
		{"NotFoundWithoutBody", http.StatusNotFound, "", http.StatusNotFound, `{"error":"Not Found"}`},
		{"ServerErrorIsBadGateway", http.StatusInternalServerError, `{"error":"boom"}`, http.StatusBadGateway, `{"error":"Failed to proxy request"}`},
		{"BadKeyIsBadGateway", http.StatusUnauthorized, `{"error":"Must be authenticated!"}`, http.StatusBadGateway, `{"error":"Failed to proxy request"}`},
		{"RateLimitIsBadGateway", http.StatusTooManyRequests, `{"error":"slow down"}`, http.StatusBadGateway, `{"error":"Failed to proxy request"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAlchemy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.alchemyCode)
				w.Write([]byte(tt.alchemyBody))
			}))
			defer mockAlchemy.Close()

			opts := httpclient.DefaultRetryOptions()
			opts.MaxRetries = 2
			opts.BaseBackoff = time.Millisecond
			client := alchemy.NewClient("test-api-key", map[string]string{"eth-mainnet": mockAlchemy.URL}, opts)
			server := NewServer(client, zap.NewNop())

			req := httptest.NewRequest(http.MethodGet, "/eth/mainnet/nft/v3/getNFTsForOwner", nil)
			req = mux.SetURLVars(req, map[string]string{"chain": "eth", "network": "mainnet"})
			w := httptest.NewRecorder()

			server.handleProxy(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}
			if got := strings.TrimSpace(w.Body.String()); got != tt.wantBody {
				t.Errorf("Expected body %s, got %s", tt.wantBody, got)
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Expected a JSON answer, got Content-Type %q", got)
			}
		})
	}
}

func TestHandleHealth(t *testing.T) {
	logger := zap.NewNop()
	client := alchemy.NewClient("test-api-key", map[string]string{}, httpclient.DefaultRetryOptions())
	server := NewServer(client, logger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %q", response["status"])
	}
}

func TestSetupRoutes(t *testing.T) {
	logger := zap.NewNop()
	client := alchemy.NewClient("test-api-key", map[string]string{}, httpclient.DefaultRetryOptions())
	server := NewServer(client, logger)

	router := mux.NewRouter()
	server.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected health endpoint to return 200, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/eth/mainnet/nft/v3/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// No chains are configured, so the handler answers 404 itself: its own
	// message tells that apart from a route the router does not know
	if !strings.Contains(w.Body.String(), "unsupported chain") {
		t.Errorf("NFT proxy route not registered: %d %q", w.Code, w.Body.String())
	}
}
