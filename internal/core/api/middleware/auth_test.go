package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAdminAuth_RequiresHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(AdminAuth(nil, nil))
	r.GET("/v1/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestAdminAuth_AcceptsAnyNonEmptyWhenNoConfiguredKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(AdminAuth(nil, nil))
	r.GET("/v1/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("X-Admin-API-Key", "manual-run")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

func TestAdminAuth_RejectsUnknownKeyWhenConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(AdminAuth([]string{"prod-key-1"}, nil))
	r.GET("/v1/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("X-Admin-API-Key", "wrong-key")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestAdminAuth_AcceptsConfiguredKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(AdminAuth([]string{"prod-key-1", "prod-key-2"}, nil))
	r.GET("/v1/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("X-Admin-API-Key", "prod-key-2")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

func TestAdminAuth_AcceptsBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(AdminAuth(nil, staticVerifier{
		token:   "valid.token",
		adminID: "master-admin",
	}))
	r.GET("/v1/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer valid.token")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

func TestAdminAuth_UsesAPIKeyWhenBearerInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(AdminAuth([]string{"prod-key-1"}, staticVerifier{
		token:   "valid.token",
		adminID: "master-admin",
	}))
	r.GET("/v1/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token")
	req.Header.Set("X-Admin-API-Key", "prod-key-1")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

func TestAdminAuth_AllowsAdminLoginWithoutHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(AdminAuth(nil, nil))
	r.POST("/v1/admin/auth/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/auth/login", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

type staticVerifier struct {
	token   string
	adminID string
}

func (s staticVerifier) Verify(token string, _ time.Time) (string, time.Time, error) {
	if token != s.token {
		return "", time.Time{}, http.ErrNoCookie
	}
	return s.adminID, time.Now().Add(time.Hour), nil
}
