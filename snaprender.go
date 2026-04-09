// Package snaprender provides a Go client for the SnapRender Screenshot API.
//
// Usage:
//
//	client := snaprender.NewClient("sk_live_...")
//	img, err := client.Capture(ctx, "https://example.com", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	os.WriteFile("screenshot.png", img, 0644)
package snaprender

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultBaseURL = "https://app.snap-render.com"
	defaultTimeout = 60 * time.Second
)

// Client is the SnapRender API client.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithBaseURL sets a custom API base URL.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) { c.baseURL = url }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.http = hc }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) { c.http.Timeout = d }
}

// NewClient creates a new SnapRender API client.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: defaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Capture takes a screenshot of a URL and returns the image bytes.
// Pass nil for opts to use defaults.
func (c *Client) Capture(ctx context.Context, urlStr string, opts *CaptureOptions) ([]byte, error) {
	if opts == nil {
		opts = &CaptureOptions{}
	}

	params := url.Values{}
	params.Set("url", urlStr)
	setCommonParams(params, opts)

	reqURL := c.baseURL + "/v1/screenshot?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	return c.doImageRequest(req)
}

// CaptureHTML renders raw HTML and returns the screenshot bytes.
func (c *Client) CaptureHTML(ctx context.Context, html string, opts *CaptureOptions) ([]byte, error) {
	return c.capturePost(ctx, "html", html, opts)
}

// CaptureMarkdown renders Markdown and returns the screenshot bytes.
func (c *Client) CaptureMarkdown(ctx context.Context, markdown string, opts *CaptureOptions) ([]byte, error) {
	return c.capturePost(ctx, "markdown", markdown, opts)
}

func (c *Client) capturePost(ctx context.Context, sourceKey, sourceValue string, opts *CaptureOptions) ([]byte, error) {
	if opts == nil {
		opts = &CaptureOptions{}
	}

	body := map[string]interface{}{sourceKey: sourceValue}
	addCommonBody(body, opts)

	return c.doPostImage(ctx, "/v1/screenshot", body)
}

// CaptureJSON takes a screenshot and returns structured JSON metadata.
func (c *Client) CaptureJSON(ctx context.Context, urlStr string, opts *CaptureOptions) (*CaptureJSONResponse, error) {
	if opts == nil {
		opts = &CaptureOptions{}
	}

	params := url.Values{}
	params.Set("url", urlStr)
	params.Set("response_type", "json")
	setCommonParams(params, opts)

	reqURL := c.baseURL + "/v1/screenshot?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result CaptureJSONResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("snaprender: failed to decode JSON response: %w", err)
	}
	return &result, nil
}

// Sign generates a pre-signed URL that can render a screenshot without an API key.
// Signing is free and does not count against quota.
func (c *Client) Sign(ctx context.Context, urlStr string, opts *SignOptions) (*SignedURLResponse, error) {
	if opts == nil {
		opts = &SignOptions{}
	}

	body := map[string]interface{}{"url": urlStr}
	if opts.ExpiresIn > 0 {
		body["expires_in"] = opts.ExpiresIn
	}
	if opts.Format != "" {
		body["format"] = opts.Format
	}
	if opts.Width > 0 {
		body["width"] = opts.Width
	}
	if opts.Height > 0 {
		body["height"] = opts.Height
	}
	if opts.FullPage != nil {
		body["full_page"] = *opts.FullPage
	}
	if opts.Quality > 0 {
		body["quality"] = opts.Quality
	}
	if opts.Delay > 0 {
		body["delay"] = opts.Delay
	}
	if opts.DarkMode != nil {
		body["dark_mode"] = *opts.DarkMode
	}
	if opts.BlockAds != nil {
		body["block_ads"] = *opts.BlockAds
	}
	if opts.BlockCookieBanners != nil {
		body["block_cookie_banners"] = *opts.BlockCookieBanners
	}
	if opts.Device != "" {
		body["device"] = opts.Device
	}
	if opts.HideSelectors != "" {
		body["hide_selectors"] = opts.HideSelectors
	}
	if opts.ClickSelector != "" {
		body["click_selector"] = opts.ClickSelector
	}
	if opts.UserAgent != "" {
		body["user_agent"] = opts.UserAgent
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/screenshot/sign", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result SignedURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("snaprender: failed to decode sign response: %w", err)
	}
	return &result, nil
}

// Extract extracts content from a web page.
// Supports types: markdown, text, html, article, links, metadata.
func (c *Client) Extract(ctx context.Context, urlStr string, opts *ExtractOptions) (*ExtractResponse, error) {
	if opts == nil {
		opts = &ExtractOptions{}
	}

	body := map[string]interface{}{"url": urlStr}
	if opts.Type != "" {
		body["type"] = opts.Type
	}
	if opts.Selector != "" {
		body["selector"] = opts.Selector
	}
	if opts.BlockAds != nil {
		body["block_ads"] = *opts.BlockAds
	}
	if opts.BlockCookieBanners != nil {
		body["block_cookie_banners"] = *opts.BlockCookieBanners
	}
	if opts.Delay > 0 {
		body["delay"] = opts.Delay
	}
	if opts.MaxLength > 0 {
		body["max_length"] = opts.MaxLength
	}
	if opts.Cache != nil {
		body["cache"] = *opts.Cache
	}
	if opts.CacheTTL > 0 {
		body["cache_ttl"] = opts.CacheTTL
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/extract", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result ExtractResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("snaprender: failed to decode extract response: %w", err)
	}
	return &result, nil
}

// Batch creates a batch screenshot job for multiple URLs (1-50).
// Returns immediately with a job ID. Poll with GetBatchStatus() for results.
func (c *Client) Batch(ctx context.Context, urls []string, opts *BatchOptions) (*BatchJobResponse, error) {
	if opts == nil {
		opts = &BatchOptions{}
	}

	body := map[string]interface{}{"urls": urls}
	if opts.Format != "" {
		body["format"] = opts.Format
	}
	if opts.Width > 0 {
		body["width"] = opts.Width
	}
	if opts.Height > 0 {
		body["height"] = opts.Height
	}
	if opts.FullPage != nil {
		body["full_page"] = *opts.FullPage
	}
	if opts.Quality > 0 {
		body["quality"] = opts.Quality
	}
	if opts.Delay > 0 {
		body["delay"] = opts.Delay
	}
	if opts.DarkMode != nil {
		body["dark_mode"] = *opts.DarkMode
	}
	if opts.BlockAds != nil {
		body["block_ads"] = *opts.BlockAds
	}
	if opts.BlockCookieBanners != nil {
		body["block_cookie_banners"] = *opts.BlockCookieBanners
	}
	if opts.Device != "" {
		body["device"] = opts.Device
	}
	if opts.HideSelectors != "" {
		body["hide_selectors"] = opts.HideSelectors
	}
	if opts.ClickSelector != "" {
		body["click_selector"] = opts.ClickSelector
	}
	if opts.UserAgent != "" {
		body["user_agent"] = opts.UserAgent
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/screenshot/batch", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, parseAPIError(resp)
	}

	var result BatchJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("snaprender: failed to decode batch response: %w", err)
	}
	return &result, nil
}

// GetBatchStatus gets the status of a batch screenshot job.
// Poll this until Status is "completed" or "failed".
func (c *Client) GetBatchStatus(ctx context.Context, jobID string) (*BatchJobResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/screenshot/batch/"+jobID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result BatchJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("snaprender: failed to decode batch status response: %w", err)
	}
	return &result, nil
}

// CreateWebhook registers a new webhook. Max 5 per account.
func (c *Client) CreateWebhook(ctx context.Context, opts *WebhookCreateOptions) (*WebhookResponse, error) {
	data, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/webhooks", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}

	var result WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("snaprender: failed to decode webhook response: %w", err)
	}
	return &result, nil
}

// ListWebhooks returns all webhooks for the authenticated account.
func (c *Client) ListWebhooks(ctx context.Context) ([]WebhookResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/webhooks", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result []WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("snaprender: failed to decode webhooks response: %w", err)
	}
	return result, nil
}

// DeleteWebhook removes a webhook by ID.
func (c *Client) DeleteWebhook(ctx context.Context, webhookID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/v1/webhooks/"+webhookID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return parseAPIError(resp)
	}
	return nil
}

// TestWebhook sends a test payload to a webhook.
func (c *Client) TestWebhook(ctx context.Context, webhookID string) (*WebhookTestResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/webhooks/"+webhookID+"/test", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result WebhookTestResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("snaprender: failed to decode webhook test response: %w", err)
	}
	return &result, nil
}

// VerifyWebhookSignature verifies an incoming webhook payload's HMAC-SHA256 signature.
func VerifyWebhookSignature(payload, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	sig := signature
	if len(sig) > 7 && sig[:7] == "sha256=" {
		sig = sig[7:]
	}
	return hmac.Equal([]byte(sig), []byte(expected))
}

// Usage returns the current month's usage statistics.
func (c *Client) Usage(ctx context.Context) (*UsageResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/usage", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result UsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("snaprender: failed to decode usage response: %w", err)
	}
	return &result, nil
}

// Info checks the cache status for a screenshot without capturing it.
// Pass nil for opts to use defaults. The cache key depends on url + all capture params.
func (c *Client) Info(ctx context.Context, urlStr string, opts *CaptureOptions) (*InfoResponse, error) {
	if opts == nil {
		opts = &CaptureOptions{}
	}

	params := url.Values{}
	params.Set("url", urlStr)
	setCommonParams(params, opts)

	reqURL := c.baseURL + "/v1/screenshot/info?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result InfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("snaprender: failed to decode info response: %w", err)
	}
	return &result, nil
}

// UsageDaily returns a daily usage breakdown for the authenticated account.
// Pass 0 for days to use the default (30). Valid range is 1-90.
func (c *Client) UsageDaily(ctx context.Context, days int) (*UsageDailyResponse, error) {
	if days <= 0 {
		days = 30
	}

	reqURL := c.baseURL + "/v1/usage/daily?days=" + strconv.Itoa(days)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result UsageDailyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("snaprender: failed to decode daily usage response: %w", err)
	}
	return &result, nil
}

// --- internal helpers ---

func (c *Client) doImageRequest(req *http.Request) ([]byte, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) doPostImage(ctx context.Context, path string, body map[string]interface{}) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	return c.doImageRequest(req)
}

func setCommonParams(params url.Values, opts *CaptureOptions) {
	if opts.Format != "" {
		params.Set("format", opts.Format)
	}
	if opts.Width > 0 {
		params.Set("width", strconv.Itoa(opts.Width))
	}
	if opts.Height > 0 {
		params.Set("height", strconv.Itoa(opts.Height))
	}
	if opts.FullPage != nil {
		params.Set("full_page", strconv.FormatBool(*opts.FullPage))
	}
	if opts.Quality > 0 {
		params.Set("quality", strconv.Itoa(opts.Quality))
	}
	if opts.Delay > 0 {
		params.Set("delay", strconv.Itoa(opts.Delay))
	}
	if opts.DarkMode != nil {
		params.Set("dark_mode", strconv.FormatBool(*opts.DarkMode))
	}
	if opts.BlockAds != nil {
		params.Set("block_ads", strconv.FormatBool(*opts.BlockAds))
	}
	if opts.BlockCookieBanners != nil {
		params.Set("block_cookie_banners", strconv.FormatBool(*opts.BlockCookieBanners))
	}
	if opts.Device != "" {
		params.Set("device", opts.Device)
	}
	if opts.HideSelectors != "" {
		params.Set("hide_selectors", opts.HideSelectors)
	}
	if opts.ClickSelector != "" {
		params.Set("click_selector", opts.ClickSelector)
	}
	if opts.UserAgent != "" {
		params.Set("user_agent", opts.UserAgent)
	}
	if opts.Cache != nil {
		params.Set("cache", strconv.FormatBool(*opts.Cache))
	}
	if opts.CacheTTL > 0 {
		params.Set("cache_ttl", strconv.Itoa(opts.CacheTTL))
	}
	if opts.ResponseType != "" {
		params.Set("response_type", opts.ResponseType)
	}
}

func addCommonBody(body map[string]interface{}, opts *CaptureOptions) {
	if opts.Format != "" {
		body["format"] = opts.Format
	}
	if opts.Width > 0 {
		body["width"] = opts.Width
	}
	if opts.Height > 0 {
		body["height"] = opts.Height
	}
	if opts.FullPage != nil {
		body["full_page"] = *opts.FullPage
	}
	if opts.Quality > 0 {
		body["quality"] = opts.Quality
	}
	if opts.Delay > 0 {
		body["delay"] = opts.Delay
	}
	if opts.DarkMode != nil {
		body["dark_mode"] = *opts.DarkMode
	}
	if opts.BlockAds != nil {
		body["block_ads"] = *opts.BlockAds
	}
	if opts.BlockCookieBanners != nil {
		body["block_cookie_banners"] = *opts.BlockCookieBanners
	}
	if opts.Device != "" {
		body["device"] = opts.Device
	}
	if opts.HideSelectors != "" {
		body["hide_selectors"] = opts.HideSelectors
	}
	if opts.ClickSelector != "" {
		body["click_selector"] = opts.ClickSelector
	}
	if opts.UserAgent != "" {
		body["user_agent"] = opts.UserAgent
	}
	if opts.Cache != nil {
		body["cache"] = *opts.Cache
	}
	if opts.CacheTTL > 0 {
		body["cache_ttl"] = opts.CacheTTL
	}
	if opts.ResponseType != "" {
		body["response_type"] = opts.ResponseType
	}
}

func parseAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var apiErr struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Status  int    `json:"status"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Code != "" {
		return &Error{
			Code:    apiErr.Error.Code,
			Message: apiErr.Error.Message,
			Status:  resp.StatusCode,
		}
	}

	return &Error{
		Code:    "UNKNOWN",
		Message: fmt.Sprintf("HTTP %d", resp.StatusCode),
		Status:  resp.StatusCode,
	}
}
