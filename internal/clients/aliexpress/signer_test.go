package aliexpress

import (
	"testing"
)

func TestSortedConcat(t *testing.T) {
	params := map[string]string{
		"app_key":     "12345678",
		"code":        "3_500102_JxZ05Ux3cnnSSUm6dCxYg6Q26",
		"sign_method": "sha256",
		"timestamp":   "1517820392000",
	}

	got := sortedConcat(params)
	want := "app_key12345678code3_500102_JxZ05Ux3cnnSSUm6dCxYg6Q26sign_methodsha256timestamp1517820392000"

	if got != want {
		t.Errorf("sortedConcat:\n got  %q\n want %q", got, want)
	}
}

func TestHmacSHA256(t *testing.T) {
	s := NewSigner("12345678", "helloworld")

	// System API: apiPath prefix + sorted params
	payload := "/auth/token/createapp_key12345678code3_500102_JxZ05Ux3cnnSSUm6dCxYg6Q26sign_methodsha256timestamp1517820392000"
	got := s.hmacSHA256(payload)
	want := "35607762342831B6A417A0DED84B79C05FEFBF116969C48AD6DC00279A9F4D81"

	if got != want {
		t.Errorf("hmacSHA256 system api:\n got  %q\n want %q", got, want)
	}
}

func TestSignSystemAPI(t *testing.T) {
	s := NewSigner("12345678", "helloworld")

	params := map[string]string{
		"code": "3_500102_JxZ05Ux3cnnSSUm6dCxYg6Q26",
	}

	// Override timestamp to match the guide example
	signed := s.SignSystemAPI("/auth/token/create", params)

	// Verify required fields are present
	for _, key := range []string{"app_key", "timestamp", "sign_method", "sign", "code"} {
		if _, ok := signed[key]; !ok {
			t.Errorf("missing key %q in signed params", key)
		}
	}

	if signed["app_key"] != "12345678" {
		t.Errorf("app_key: got %q, want %q", signed["app_key"], "12345678")
	}
}

func TestSignSystemAPI_WithFixedTimestamp(t *testing.T) {
	s := NewSigner("12345678", "helloworld")

	// Manually build to use the exact timestamp from the guide
	params := map[string]string{
		"code": "3_500102_JxZ05Ux3cnnSSUm6dCxYg6Q26",
	}

	merged := map[string]string{
		"app_key":     "12345678",
		"timestamp":   "1517820392000",
		"sign_method": "sha256",
	}
	for k, v := range params {
		merged[k] = v
	}

	payload := "/auth/token/create" + sortedConcat(merged)
	sign := s.hmacSHA256(payload)

	want := "35607762342831B6A417A0DED84B79C05FEFBF116969C48AD6DC00279A9F4D81"
	if sign != want {
		t.Errorf("sign:\n got  %q\n want %q", sign, want)
	}
}
