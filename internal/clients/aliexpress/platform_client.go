package aliexpress

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

func (c *PlatformClient) AppKey() string {
	if c == nil || c.signer == nil {
		return ""
	}
	return c.signer.appKey
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

	variants := []systemAPIVariant{
		{name: "rest-post-body", basePath: "/rest", method: http.MethodPost, bizInBody: true},
		{name: "rest-get-query", basePath: "/rest", method: http.MethodGet, bizInBody: false},
	}

	var lastResp *PlatformResponse
	var lastErr error
	for _, variant := range variants {
		resp, err := c.callSystemAPIVariant(ctx, apiPath, params, variant)
		if err != nil {
			c.logSystemAPIDebug(apiPath, params, variant, "", err)
			lastErr = err
			continue
		}
		if !isIncompleteSignature(resp) {
			return resp, nil
		}
		c.logSystemAPIDebug(apiPath, params, variant, resp.Code, nil)
		lastResp = resp
	}
	if lastResp != nil {
		return lastResp, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("call system api failed: no variant succeeded")
}

type systemAPIVariant struct {
	name      string
	basePath  string
	method    string
	bizInBody bool
}

func (c *PlatformClient) callSystemAPIVariant(ctx context.Context, apiPath string, params map[string]string, variant systemAPIVariant) (*PlatformResponse, error) {
	commonParams := c.signer.commonParams()
	allParams := make(map[string]string, len(commonParams)+len(params)+1)
	for k, v := range commonParams {
		allParams[k] = v
	}
	for k, v := range params {
		allParams[k] = v
	}
	commonParams["sign"] = c.signer.hmacSHA256(apiPath + sortedConcat(allParams))

	query := url.Values{}
	for k, v := range commonParams {
		query.Set(k, v)
	}
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	reqURL := c.baseURL + variant.basePath + apiPath
	var body *bytes.Buffer
	if variant.bizInBody {
		body = bytes.NewBufferString(form.Encode())
	} else {
		body = bytes.NewBuffer(nil)
		for k, values := range form {
			for _, v := range values {
				query.Set(k, v)
			}
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, variant.method, reqURL+"?"+query.Encode(), body)
	if err != nil {
		return nil, fmt.Errorf("build system api request (%s): %w", variant.name, err)
	}
	if variant.method == http.MethodPost {
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return c.doRequest(httpReq)
}

func isIncompleteSignature(resp *PlatformResponse) bool {
	if resp == nil {
		return false
	}
	return resp.Code == "IncompleteSignature"
}

func (c *PlatformClient) logSystemAPIDebug(apiPath string, params map[string]string, variant systemAPIVariant, code string, err error) {
	if !strings.Contains(apiPath, "/auth/token/") {
		return
	}

	signPayload := apiPath + sortedConcat(mergeMaps(c.signer.commonParams(), params))
	message := fmt.Sprintf(
		"system api debug: path=%s variant=%s method=%s biz_in_body=%t app_key=%s refresh_token=%s payload=%s",
		apiPath,
		variant.name,
		variant.method,
		variant.bizInBody,
		fingerprintText(c.signer.appKey),
		fingerprintText(params["refresh_token"]),
		fingerprintText(signPayload),
	)
	if err != nil {
		log.Printf("%s err=%v", message, err)
		return
	}
	log.Printf("%s resp_code=%s", message, code)
}

func mergeMaps(a map[string]string, b map[string]string) map[string]string {
	merged := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		merged[k] = v
	}
	for k, v := range b {
		merged[k] = v
	}
	return merged
}

func fingerprintText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("len=%d sha256=%s", len(value), hex.EncodeToString(sum[:4]))
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
