package snaprender

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// recorded is what the fake API saw for the last request.
type recorded struct {
	method string
	path   string
	query  map[string]string
	body   map[string]interface{}
	apiKey string
}

// fakeAPI answers every request with the given status and body and records the request.
func fakeAPI(t *testing.T, status int, respBody string) (*httptest.Server, *recorded) {
	t.Helper()
	rec := &recorded{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.method = r.Method
		rec.path = r.URL.Path
		rec.apiKey = r.Header.Get("X-API-Key")
		rec.query = map[string]string{}
		for k, v := range r.URL.Query() {
			rec.query[k] = v[0]
		}
		rec.body = nil
		if data, _ := io.ReadAll(r.Body); len(data) > 0 {
			_ = json.Unmarshal(data, &rec.body)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

func allCaptureOptions() *CaptureOptions {
	return &CaptureOptions{
		Format: "webp", Width: 1440, Height: 900, FullPage: Bool(true), Quality: 80, Delay: 500,
		DarkMode: Bool(true), BlockAds: Bool(true), BlockCookieBanners: Bool(false), Device: "iphone_15",
		HideSelectors: ".chat", ClickSelector: "#accept", UserAgent: "test-agent", Cache: Bool(false),
		CacheTTL: 3600, ResponseType: "binary",
	}
}

func expectKeys(t *testing.T, got map[string]interface{}, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if _, ok := got[k]; !ok {
			t.Errorf("expected key %q in request body, got %v", k, got)
		}
	}
}

func TestClientOptions(t *testing.T) {
	hc := &http.Client{}
	c := NewClient("k", WithHTTPClient(hc), WithTimeout(5*time.Second))
	if c.http != hc {
		t.Fatal("WithHTTPClient did not set the client")
	}
	if c.http.Timeout != 5*time.Second {
		t.Fatalf("WithTimeout did not set the timeout, got %v", c.http.Timeout)
	}
}

func TestIsNotFound(t *testing.T) {
	if !(&Error{Status: 404}).IsNotFound() {
		t.Fatal("expected IsNotFound for 404")
	}
	if (&Error{Status: 500}).IsNotFound() {
		t.Fatal("did not expect IsNotFound for 500")
	}
}

func TestCaptureAllQueryParams(t *testing.T) {
	srv, rec := fakeAPI(t, 200, "img")
	c := NewClient("k", WithBaseURL(srv.URL))
	if _, err := c.Capture(context.Background(), "https://example.com", allCaptureOptions()); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"format", "width", "height", "full_page", "quality", "delay", "dark_mode", "block_ads",
		"block_cookie_banners", "device", "hide_selectors", "click_selector", "user_agent", "cache", "cache_ttl", "response_type"} {
		if _, ok := rec.query[k]; !ok {
			t.Errorf("missing query param %q", k)
		}
	}
	if rec.query["block_cookie_banners"] != "false" || rec.query["cache"] != "false" {
		t.Errorf("explicit false values must be sent, got %v", rec.query)
	}
}

func TestCaptureHTMLAllBodyFields(t *testing.T) {
	srv, rec := fakeAPI(t, 200, "img")
	c := NewClient("k", WithBaseURL(srv.URL))
	if _, err := c.CaptureHTML(context.Background(), "<h1>hi</h1>", allCaptureOptions()); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/v1/screenshot" {
		t.Fatalf("unexpected request %s %s", rec.method, rec.path)
	}
	expectKeys(t, rec.body, "html", "format", "width", "height", "full_page", "quality", "delay", "dark_mode", "block_ads",
		"block_cookie_banners", "device", "hide_selectors", "click_selector", "user_agent", "cache", "cache_ttl", "response_type")
}

func TestCaptureJSONWithOptions(t *testing.T) {
	srv, rec := fakeAPI(t, 200, `{"format":"png","width":1280,"height":800,"image":"data:image/png;base64,AA==","size":1,"cache":"MISS"}`)
	c := NewClient("k", WithBaseURL(srv.URL))
	res, err := c.CaptureJSON(context.Background(), "https://example.com", &CaptureOptions{Width: 1280})
	if err != nil {
		t.Fatal(err)
	}
	if rec.query["response_type"] != "json" || rec.query["width"] != "1280" || res.Width != 1280 || res.Cache != "MISS" {
		t.Fatalf("unexpected result %+v, query %v", res, rec.query)
	}
}

func TestSignAllFields(t *testing.T) {
	srv, rec := fakeAPI(t, 200, `{"signed_url":"https://app.snap-render.com/s/abc","expires_at":"2030-01-01T00:00:00Z","expires_in":3600}`)
	c := NewClient("k", WithBaseURL(srv.URL))
	res, err := c.Sign(context.Background(), "https://example.com", &SignOptions{
		ExpiresIn: 3600, Format: "png", Width: 1200, Height: 630, FullPage: Bool(false), Quality: 90, Delay: 100,
		DarkMode: Bool(true), BlockAds: Bool(true), BlockCookieBanners: Bool(true), Device: "desktop",
		HideSelectors: ".x", ClickSelector: ".y", UserAgent: "ua",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/v1/screenshot/sign" || res.ExpiresIn != 3600 {
		t.Fatalf("unexpected path %s or result %+v", rec.path, res)
	}
	expectKeys(t, rec.body, "url", "expires_in", "format", "width", "height", "full_page", "quality", "delay", "dark_mode",
		"block_ads", "block_cookie_banners", "device", "hide_selectors", "click_selector", "user_agent")
}

func TestExtract(t *testing.T) {
	srv, rec := fakeAPI(t, 200, `{"url":"https://example.com","type":"markdown","content":"# Example","wordCount":1,"processingTimeMs":42}`)
	c := NewClient("k", WithBaseURL(srv.URL))
	res, err := c.Extract(context.Background(), "https://example.com", &ExtractOptions{
		Type: "markdown", Selector: "main", BlockAds: Bool(true), BlockCookieBanners: Bool(true), Delay: 200,
		MaxLength: 5000, Cache: Bool(false), CacheTTL: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/v1/extract" || rec.apiKey != "k" {
		t.Fatalf("unexpected request %s %s key=%q", rec.method, rec.path, rec.apiKey)
	}
	expectKeys(t, rec.body, "url", "type", "selector", "block_ads", "block_cookie_banners", "delay", "max_length", "cache", "cache_ttl")
	var content string
	if err := json.Unmarshal(res.Content, &content); err != nil || content != "# Example" {
		t.Fatalf("unexpected content %s (%v)", res.Content, err)
	}
	if res.WordCount == nil || *res.WordCount != 1 || res.ProcessingTimeMs != 42 {
		t.Fatalf("unexpected result %+v", res)
	}
}

func TestExtractDefaults(t *testing.T) {
	srv, rec := fakeAPI(t, 200, `{"url":"https://example.com","type":"markdown","content":""}`)
	c := NewClient("k", WithBaseURL(srv.URL))
	if _, err := c.Extract(context.Background(), "https://example.com", nil); err != nil {
		t.Fatal(err)
	}
	if len(rec.body) != 1 {
		t.Fatalf("nil options should send only the url, got %v", rec.body)
	}
}

func TestBatch(t *testing.T) {
	srv, rec := fakeAPI(t, 202, `{"jobId":"job_1","status":"pending","statusUrl":"/v1/screenshot/batch/job_1","total":2}`)
	c := NewClient("k", WithBaseURL(srv.URL))
	res, err := c.Batch(context.Background(), []string{"https://a.com", "https://b.com"}, &BatchOptions{
		Format: "jpeg", Width: 1024, Height: 768, FullPage: Bool(true), Quality: 70, Delay: 50, DarkMode: Bool(false),
		BlockAds: Bool(true), BlockCookieBanners: Bool(true), Device: "ipad", HideSelectors: ".a", ClickSelector: ".b", UserAgent: "ua",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/v1/screenshot/batch" || res.JobID != "job_1" || res.Total != 2 {
		t.Fatalf("unexpected path %s or result %+v", rec.path, res)
	}
	expectKeys(t, rec.body, "urls", "format", "width", "height", "full_page", "quality", "delay", "dark_mode", "block_ads",
		"block_cookie_banners", "device", "hide_selectors", "click_selector", "user_agent")
	if urls, _ := rec.body["urls"].([]interface{}); len(urls) != 2 {
		t.Fatalf("expected 2 urls, got %v", rec.body["urls"])
	}
}

func TestBatchDefaults(t *testing.T) {
	srv, rec := fakeAPI(t, 200, `{"jobId":"job_2","status":"pending"}`)
	c := NewClient("k", WithBaseURL(srv.URL))
	if _, err := c.Batch(context.Background(), []string{"https://a.com"}, nil); err != nil {
		t.Fatal(err)
	}
	if len(rec.body) != 1 {
		t.Fatalf("nil options should send only urls, got %v", rec.body)
	}
}

func TestGetBatchStatus(t *testing.T) {
	srv, rec := fakeAPI(t, 200, `{"jobId":"job_1","status":"completed","total":1,"completed":1,"items":[{"url":"https://a.com","status":"completed","downloadUrl":"https://cdn/x.png"}]}`)
	c := NewClient("k", WithBaseURL(srv.URL))
	res, err := c.GetBatchStatus(context.Background(), "job_1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodGet || rec.path != "/v1/screenshot/batch/job_1" {
		t.Fatalf("unexpected request %s %s", rec.method, rec.path)
	}
	if res.Status != "completed" || len(res.Items) != 1 || res.Items[0].DownloadURL != "https://cdn/x.png" {
		t.Fatalf("unexpected result %+v", res)
	}
}

func TestCreateWebhook(t *testing.T) {
	srv, rec := fakeAPI(t, 201, `{"id":"wh_1","url":"https://hooks.example.com","events":["screenshot.completed"],"secret":"s","isActive":true}`)
	c := NewClient("k", WithBaseURL(srv.URL))
	res, err := c.CreateWebhook(context.Background(), &WebhookCreateOptions{URL: "https://hooks.example.com", Events: []string{"screenshot.completed"}})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/v1/webhooks" || rec.body["url"] != "https://hooks.example.com" {
		t.Fatalf("unexpected request %s %s %v", rec.method, rec.path, rec.body)
	}
	if res.ID != "wh_1" || !res.IsActive {
		t.Fatalf("unexpected result %+v", res)
	}
}

func TestListWebhooks(t *testing.T) {
	srv, rec := fakeAPI(t, 200, `[{"id":"wh_1"},{"id":"wh_2"}]`)
	c := NewClient("k", WithBaseURL(srv.URL))
	res, err := c.ListWebhooks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodGet || rec.path != "/v1/webhooks" || len(res) != 2 || res[1].ID != "wh_2" {
		t.Fatalf("unexpected request %s %s or result %+v", rec.method, rec.path, res)
	}
}

func TestDeleteWebhook(t *testing.T) {
	for _, status := range []int{200, 204} {
		srv, rec := fakeAPI(t, status, "")
		c := NewClient("k", WithBaseURL(srv.URL))
		if err := c.DeleteWebhook(context.Background(), "wh_1"); err != nil {
			t.Fatalf("status %d: %v", status, err)
		}
		if rec.method != http.MethodDelete || rec.path != "/v1/webhooks/wh_1" {
			t.Fatalf("unexpected request %s %s", rec.method, rec.path)
		}
	}
}

func TestTestWebhook(t *testing.T) {
	srv, rec := fakeAPI(t, 200, `{"deliveryId":"d_1","statusCode":200,"success":true,"deliveredAt":"2026-01-01T00:00:00Z"}`)
	c := NewClient("k", WithBaseURL(srv.URL))
	res, err := c.TestWebhook(context.Background(), "wh_1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/v1/webhooks/wh_1/test" {
		t.Fatalf("unexpected request %s %s", rec.method, rec.path)
	}
	if !res.Success || res.StatusCode == nil || *res.StatusCode != 200 {
		t.Fatalf("unexpected result %+v", res)
	}
}

func TestVerifyWebhookSignature(t *testing.T) {
	payload, secret := `{"event":"screenshot.completed"}`, "whsec_test"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	if !VerifyWebhookSignature(payload, sig, secret) {
		t.Error("valid bare signature rejected")
	}
	if !VerifyWebhookSignature(payload, "sha256="+sig, secret) {
		t.Error("valid prefixed signature rejected")
	}
	if VerifyWebhookSignature(payload, sig, "wrong-secret") {
		t.Error("signature accepted with the wrong secret")
	}
	if VerifyWebhookSignature(payload+" ", sig, secret) {
		t.Error("signature accepted for a modified payload")
	}
	if VerifyWebhookSignature(payload, "", secret) {
		t.Error("empty signature accepted")
	}
}

func TestInfo(t *testing.T) {
	srv, rec := fakeAPI(t, 200, `{"url":"https://example.com","cached":true,"cache_key":"abc","content_type":"image/png"}`)
	c := NewClient("k", WithBaseURL(srv.URL))
	res, err := c.Info(context.Background(), "https://example.com", &CaptureOptions{Format: "png", FullPage: Bool(true)})
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/v1/screenshot/info" || rec.query["url"] != "https://example.com" || rec.query["full_page"] != "true" {
		t.Fatalf("unexpected request %s %v", rec.path, rec.query)
	}
	if !res.Cached || res.CacheKey != "abc" {
		t.Fatalf("unexpected result %+v", res)
	}
	if _, err := c.Info(context.Background(), "https://example.com", nil); err != nil {
		t.Fatalf("nil options: %v", err)
	}
}

func TestUsageDaily(t *testing.T) {
	srv, rec := fakeAPI(t, 200, `{"days":7,"data":[{"date":"2026-01-01","count":3}]}`)
	c := NewClient("k", WithBaseURL(srv.URL))
	res, err := c.UsageDaily(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if rec.path != "/v1/usage/daily" || rec.query["days"] != "7" || res.Days != 7 || res.Data[0].Count != 3 {
		t.Fatalf("unexpected request %s %v or result %+v", rec.path, rec.query, res)
	}
	if _, err := c.UsageDaily(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if rec.query["days"] != "30" {
		t.Fatalf("days=0 should default to 30, got %s", rec.query["days"])
	}
}

// Every endpoint must turn a structured API error into *Error with the API's code.
func TestAllEndpointsReturnAPIError(t *testing.T) {
	srv, _ := fakeAPI(t, 403, `{"error":{"code":"FORBIDDEN","message":"no","status":403}}`)
	c := NewClient("k", WithBaseURL(srv.URL))
	ctx := context.Background()
	calls := map[string]func() error{
		"Capture":        func() error { _, err := c.Capture(ctx, "https://a.com", nil); return err },
		"CaptureHTML":    func() error { _, err := c.CaptureHTML(ctx, "<p>", nil); return err },
		"CaptureJSON":    func() error { _, err := c.CaptureJSON(ctx, "https://a.com", nil); return err },
		"Sign":           func() error { _, err := c.Sign(ctx, "https://a.com", nil); return err },
		"Extract":        func() error { _, err := c.Extract(ctx, "https://a.com", nil); return err },
		"Batch":          func() error { _, err := c.Batch(ctx, []string{"https://a.com"}, nil); return err },
		"GetBatchStatus": func() error { _, err := c.GetBatchStatus(ctx, "j"); return err },
		"CreateWebhook":  func() error { _, err := c.CreateWebhook(ctx, &WebhookCreateOptions{URL: "https://h"}); return err },
		"ListWebhooks":   func() error { _, err := c.ListWebhooks(ctx); return err },
		"DeleteWebhook":  func() error { return c.DeleteWebhook(ctx, "w") },
		"TestWebhook":    func() error { _, err := c.TestWebhook(ctx, "w"); return err },
		"Usage":          func() error { _, err := c.Usage(ctx); return err },
		"Info":           func() error { _, err := c.Info(ctx, "https://a.com", nil); return err },
		"UsageDaily":     func() error { _, err := c.UsageDaily(ctx, 1); return err },
	}
	for name, call := range calls {
		var apiErr *Error
		if err := call(); !errors.As(err, &apiErr) || apiErr.Code != "FORBIDDEN" || apiErr.Status != 403 {
			t.Errorf("%s: expected FORBIDDEN *Error, got %v", name, err)
		}
	}
}

// A non-JSON error body still yields *Error with the HTTP status.
func TestUnstructuredErrorBody(t *testing.T) {
	srv, _ := fakeAPI(t, 502, "Bad Gateway")
	c := NewClient("k", WithBaseURL(srv.URL))
	_, err := c.Usage(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Code != "UNKNOWN" || apiErr.Status != 502 || !strings.Contains(apiErr.Message, "502") {
		t.Fatalf("expected UNKNOWN 502 error, got %v", err)
	}
}

// A 200 response that is not valid JSON is reported as a decode error, not a panic or empty result.
func TestJSONEndpointsRejectMalformedBody(t *testing.T) {
	srv, _ := fakeAPI(t, 200, "not json")
	c := NewClient("k", WithBaseURL(srv.URL))
	ctx := context.Background()
	calls := map[string]func() error{
		"CaptureJSON":    func() error { _, err := c.CaptureJSON(ctx, "https://a.com", nil); return err },
		"Sign":           func() error { _, err := c.Sign(ctx, "https://a.com", nil); return err },
		"Extract":        func() error { _, err := c.Extract(ctx, "https://a.com", nil); return err },
		"Batch":          func() error { _, err := c.Batch(ctx, []string{"https://a.com"}, nil); return err },
		"GetBatchStatus": func() error { _, err := c.GetBatchStatus(ctx, "j"); return err },
		"CreateWebhook":  func() error { _, err := c.CreateWebhook(ctx, &WebhookCreateOptions{}); return err },
		"ListWebhooks":   func() error { _, err := c.ListWebhooks(ctx); return err },
		"TestWebhook":    func() error { _, err := c.TestWebhook(ctx, "w"); return err },
		"Usage":          func() error { _, err := c.Usage(ctx); return err },
		"Info":           func() error { _, err := c.Info(ctx, "https://a.com", nil); return err },
		"UsageDaily":     func() error { _, err := c.UsageDaily(ctx, 1); return err },
	}
	for name, call := range calls {
		if err := call(); err == nil || !strings.Contains(err.Error(), "decode") {
			t.Errorf("%s: expected a decode error, got %v", name, err)
		}
	}
}

// Transport failures (server unreachable) and invalid base URLs surface as errors on every endpoint.
func TestTransportAndRequestErrors(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL := dead.URL
	dead.Close()
	ctx := context.Background()
	for _, base := range []string{deadURL, "http://bad host\x7f"} {
		c := NewClient("k", WithBaseURL(base), WithTimeout(2*time.Second))
		calls := map[string]func() error{
			"Capture":        func() error { _, err := c.Capture(ctx, "https://a.com", nil); return err },
			"CaptureHTML":    func() error { _, err := c.CaptureHTML(ctx, "<p>", nil); return err },
			"CaptureJSON":    func() error { _, err := c.CaptureJSON(ctx, "https://a.com", nil); return err },
			"Sign":           func() error { _, err := c.Sign(ctx, "https://a.com", nil); return err },
			"Extract":        func() error { _, err := c.Extract(ctx, "https://a.com", nil); return err },
			"Batch":          func() error { _, err := c.Batch(ctx, []string{"https://a.com"}, nil); return err },
			"GetBatchStatus": func() error { _, err := c.GetBatchStatus(ctx, "j"); return err },
			"CreateWebhook":  func() error { _, err := c.CreateWebhook(ctx, &WebhookCreateOptions{}); return err },
			"ListWebhooks":   func() error { _, err := c.ListWebhooks(ctx); return err },
			"DeleteWebhook":  func() error { return c.DeleteWebhook(ctx, "w") },
			"TestWebhook":    func() error { _, err := c.TestWebhook(ctx, "w"); return err },
			"Usage":          func() error { _, err := c.Usage(ctx); return err },
			"Info":           func() error { _, err := c.Info(ctx, "https://a.com", nil); return err },
			"UsageDaily":     func() error { _, err := c.UsageDaily(ctx, 1); return err },
		}
		for name, call := range calls {
			if err := call(); err == nil {
				t.Errorf("%s with base %q: expected an error", name, base)
			}
		}
	}
}
