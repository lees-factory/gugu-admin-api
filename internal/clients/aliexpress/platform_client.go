package aliexpress

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api-sg.aliexpress.com"

type PlatformConfig struct {
	AppKey     string
	AppSecret  string
	BaseURL    string
	HTTPClient *http.Client
}

type PlatformClient struct {
	signer     *Signer
	baseURL    string
	httpClient *http.Client
}

func NewPlatformClient(cfg PlatformConfig) *PlatformClient {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &PlatformClient{
		signer:     NewSigner(cfg.AppKey, cfg.AppSecret),
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// PlatformResponse is the top-level response envelope from AliExpress Open Platform.
type PlatformResponse struct {
	Code      string          `json:"code"`
	Type      string          `json:"type"`
	Message   string          `json:"message"`
	RequestID string          `json:"request_id"`
	Result    json.RawMessage `json:"result"`
	RawBody   string          `json:"-"`
}

func (r *PlatformResponse) IsSuccess() bool {
	return r.Code == "0" || r.Code == ""
}

// CallBusinessAPI calls a Business API via /sync endpoint.
// method: e.g. "aliexpress.affiliate.hotproduct.query"
// accessToken: required for most business APIs.
func (c *PlatformClient) CallBusinessAPI(ctx context.Context, method string, params map[string]string, accessToken string) (*PlatformResponse, error) {
	if params == nil {
		params = make(map[string]string)
	}
	if accessToken != "" {
		params["access_token"] = accessToken
	}

	signed := c.signer.SignBusinessAPI(method, params)

	reqURL := c.baseURL + "/sync"
	q := url.Values{}
	for k, v := range signed {
		q.Set(k, v)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("build business api request: %w", err)
	}

	return c.doRequest(httpReq)
}

// CallSystemAPI calls a System API via /rest endpoint.
// apiPath: e.g. "/auth/token/create"
func (c *PlatformClient) CallSystemAPI(ctx context.Context, apiPath string, params map[string]string) (*PlatformResponse, error) {
	if params == nil {
		params = make(map[string]string)
	}

	signed := c.signer.SignSystemAPI(apiPath, params)

	reqURL := c.baseURL + "/rest" + apiPath
	q := url.Values{}
	for k, v := range signed {
		q.Set(k, v)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("build system api request: %w", err)
	}

	return c.doRequest(httpReq)
}

func (c *PlatformClient) doRequest(req *http.Request) (*PlatformResponse, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result PlatformResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	result.RawBody = string(body)

	return &result, nil
}
