package aliexpress

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Signer struct {
	appKey    string
	appSecret string
}

func NewSigner(appKey, appSecret string) *Signer {
	return &Signer{
		appKey:    appKey,
		appSecret: appSecret,
	}
}

// SignBusinessAPI signs a Business API request.
// Signature input: sorted(key+value) pairs concatenated, then HMAC-SHA256 with appSecret.
func (s *Signer) SignBusinessAPI(method string, params map[string]string) map[string]string {
	merged := s.commonParams()
	for k, v := range params {
		merged[k] = v
	}
	merged["method"] = method

	payload := sortedConcat(merged)
	merged["sign"] = s.hmacSHA256(payload)

	return merged
}

// SignSystemAPI signs a System API request.
// Signature input: apiPath + sorted(key+value) pairs concatenated, then HMAC-SHA256 with appSecret.
func (s *Signer) SignSystemAPI(apiPath string, params map[string]string) map[string]string {
	merged := s.commonParams()
	for k, v := range params {
		merged[k] = v
	}

	payload := apiPath + sortedConcat(merged)
	merged["sign"] = s.hmacSHA256(payload)

	return merged
}

func (s *Signer) commonParams() map[string]string {
	return map[string]string{
		"app_key":     s.appKey,
		"timestamp":   fmt.Sprintf("%d", time.Now().UnixMilli()),
		"partner_id":  "iop-sdk-java",
		"sign_method": "sha256",
	}
}

func sortedConcat(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(params[k])
	}
	return b.String()
}

func (s *Signer) hmacSHA256(data string) string {
	h := hmac.New(sha256.New, []byte(s.appSecret))
	h.Write([]byte(data))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}
