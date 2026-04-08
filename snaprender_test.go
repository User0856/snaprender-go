package snaprender

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("sk_live_test")
	if c.apiKey != "sk_live_test" {
		t.Fatalf("expected apiKey sk_live_test, got %s", c.apiKey)
	}
	if c.baseURL != defaultBaseURL {
		t.Fatalf("expected default baseURL, got %s", c.baseURL)
	}
}

func TestNewClientWithOptions(t *testing.T) {
	c := NewClient("key", WithBaseURL("https://custom.com"))
	if c.baseURL != "https://custom.com" {
		t.Fatalf("expected custom baseURL, got %s", c.baseURL)
	}
}

func TestCapture(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Fatalf("expected X-API-Key test-key, got %s", r.Header.Get("X-API-Key"))
		}
		if r.URL.Query().Get("url") != "https://example.com" {
			t.Fatalf("expected url param, got %s", r.URL.Query().Get("url"))
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("fake-png-data"))
	}))
	defer srv.Close()

	c := NewClient("test-key", WithBaseURL(srv.URL))
	img, err := c.Capture(context.Background(), "https://example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(img) != "fake-png-data" {
		t.Fatalf("unexpected image data: %s", img)
	}
}

func TestCaptureWithOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("format") != "jpeg" {
			t.Fatalf("expected format=jpeg, got %s", q.Get("format"))
		}
		if q.Get("width") != "1920" {
			t.Fatalf("expected width=1920, got %s", q.Get("width"))
		}
		if q.Get("dark_mode") != "true" {
			t.Fatalf("expected dark_mode=true, got %s", q.Get("dark_mode"))
		}
		if q.Get("device") != "iphone_14" {
			t.Fatalf("expected device=iphone_14, got %s", q.Get("device"))
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Capture(context.Background(), "https://example.com", &CaptureOptions{
		Format:   "jpeg",
		Width:    1920,
		DarkMode: Bool(true),
		Device:   "iphone_14",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCaptureHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected JSON content type")
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["html"] != "<h1>Hello</h1>" {
			t.Fatalf("expected html in body, got %v", body)
		}
		w.Write([]byte("html-screenshot"))
	}))
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	img, err := c.CaptureHTML(context.Background(), "<h1>Hello</h1>", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(img) != "html-screenshot" {
		t.Fatalf("unexpected data: %s", img)
	}
}

func TestCaptureMarkdown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["markdown"] != "# Hello" {
			t.Fatalf("expected markdown in body, got %v", body)
		}
		w.Write([]byte("md-screenshot"))
	}))
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	img, err := c.CaptureMarkdown(context.Background(), "# Hello", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(img) != "md-screenshot" {
		t.Fatalf("unexpected data: %s", img)
	}
}

func TestCaptureJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("response_type") != "json" {
			t.Fatalf("expected response_type=json")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"url":              "https://example.com",
			"format":           "png",
			"width":            1280,
			"height":           800,
			"image":            "data:image/png;base64,abc",
			"size":             1234,
			"cache":            "MISS",
			"responseTime":     "500ms",
			"remainingCredits": 99,
		})
	}))
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	result, err := c.CaptureJSON(context.Background(), "https://example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != "png" {
		t.Fatalf("expected png, got %s", result.Format)
	}
	if result.RemainingCredits != 99 {
		t.Fatalf("expected 99 credits, got %d", result.RemainingCredits)
	}
}

func TestSign(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/screenshot/sign" {
			t.Fatalf("expected /v1/screenshot/sign, got %s", r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["url"] != "https://example.com" {
			t.Fatalf("expected url in body")
		}
		if body["expires_in"] != float64(3600) {
			t.Fatalf("expected expires_in=3600, got %v", body["expires_in"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"signed_url": "https://api.example.com/v1/screenshot/render?sig=abc",
			"expires_at": "2026-04-09T10:00:00.000Z",
			"expires_in": 3600,
		})
	}))
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	result, err := c.Sign(context.Background(), "https://example.com", &SignOptions{ExpiresIn: 3600})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExpiresIn != 3600 {
		t.Fatalf("expected 3600, got %d", result.ExpiresIn)
	}
	if result.SignedURL == "" {
		t.Fatal("expected signed_url")
	}
}

func TestUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/usage" {
			t.Fatalf("expected /v1/usage, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"plan": "growth",
			"period": map[string]string{
				"start": "2026-04-01T00:00:00.000Z",
				"end":   "2026-04-30T23:59:59.000Z",
			},
			"usage": map[string]int{
				"screenshots_used":      500,
				"screenshots_limit":     10000,
				"screenshots_remaining": 9500,
			},
		})
	}))
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	result, err := c.Usage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Plan != "growth" {
		t.Fatalf("expected growth, got %s", result.Plan)
	}
	if result.Usage.ScreenshotsRemaining != 9500 {
		t.Fatalf("expected 9500, got %d", result.Usage.ScreenshotsRemaining)
	}
}

func TestAPIError401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Missing or invalid API key.",
				"status":  401,
			},
		})
	}))
	defer srv.Close()

	c := NewClient("bad-key", WithBaseURL(srv.URL))
	_, err := c.Capture(context.Background(), "https://example.com", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if !apiErr.IsUnauthorized() {
		t.Fatalf("expected unauthorized, got status %d", apiErr.Status)
	}
	if apiErr.Code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED, got %s", apiErr.Code)
	}
}

func TestAPIError429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(429)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "QUOTA_EXCEEDED",
				"message": "Monthly quota exceeded.",
				"status":  429,
			},
		})
	}))
	defer srv.Close()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Capture(context.Background(), "https://example.com", nil)
	apiErr := err.(*Error)
	if !apiErr.IsRateLimited() {
		t.Fatalf("expected rate limited, got status %d", apiErr.Status)
	}
}

func TestBoolHelper(t *testing.T) {
	b := Bool(true)
	if *b != true {
		t.Fatal("expected true")
	}
	b = Bool(false)
	if *b != false {
		t.Fatal("expected false")
	}
}

func TestErrorString(t *testing.T) {
	e := &Error{Code: "TEST", Message: "test message", Status: 400}
	expected := "snaprender: TEST (400): test message"
	if e.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, e.Error())
	}
}
