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
