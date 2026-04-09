package snaprender

import "encoding/json"

// CaptureOptions configures a screenshot capture request.
type CaptureOptions struct {
	Format             string `json:"format,omitempty"`
	Width              int    `json:"width,omitempty"`
	Height             int    `json:"height,omitempty"`
	FullPage           *bool  `json:"full_page,omitempty"`
	Quality            int    `json:"quality,omitempty"`
	Delay              int    `json:"delay,omitempty"`
	DarkMode           *bool  `json:"dark_mode,omitempty"`
	BlockAds           *bool  `json:"block_ads,omitempty"`
	BlockCookieBanners *bool  `json:"block_cookie_banners,omitempty"`
	Device             string `json:"device,omitempty"`
	HideSelectors      string `json:"hide_selectors,omitempty"`
	ClickSelector      string `json:"click_selector,omitempty"`
	UserAgent          string `json:"user_agent,omitempty"`
	Cache              *bool  `json:"cache,omitempty"`
	CacheTTL           int    `json:"cache_ttl,omitempty"`
	ResponseType       string `json:"response_type,omitempty"`
}

// SignOptions configures a signed URL generation request.
type SignOptions struct {
	ExpiresIn          int    `json:"expires_in,omitempty"`
	Format             string `json:"format,omitempty"`
	Width              int    `json:"width,omitempty"`
	Height             int    `json:"height,omitempty"`
	FullPage           *bool  `json:"full_page,omitempty"`
	Quality            int    `json:"quality,omitempty"`
	Delay              int    `json:"delay,omitempty"`
	DarkMode           *bool  `json:"dark_mode,omitempty"`
	BlockAds           *bool  `json:"block_ads,omitempty"`
	BlockCookieBanners *bool  `json:"block_cookie_banners,omitempty"`
	Device             string `json:"device,omitempty"`
	HideSelectors      string `json:"hide_selectors,omitempty"`
	ClickSelector      string `json:"click_selector,omitempty"`
	UserAgent          string `json:"user_agent,omitempty"`
}

// Bool returns a pointer to a bool value, for use with option fields.
func Bool(v bool) *bool { return &v }

// ExtractOptions configures a content extraction request.
type ExtractOptions struct {
	Type               string `json:"type,omitempty"`
	Selector           string `json:"selector,omitempty"`
	BlockAds           *bool  `json:"block_ads,omitempty"`
	BlockCookieBanners *bool  `json:"block_cookie_banners,omitempty"`
	Delay              int    `json:"delay,omitempty"`
	MaxLength          int    `json:"max_length,omitempty"`
	Cache              *bool  `json:"cache,omitempty"`
	CacheTTL           int    `json:"cache_ttl,omitempty"`
}

// ExtractResponse is the response from the extract endpoint.
type ExtractResponse struct {
	URL             string          `json:"url"`
	Type            string          `json:"type"`
	Content         json.RawMessage `json:"content"`
	WordCount       *int            `json:"wordCount,omitempty"`
	ProcessingTimeMs int            `json:"processingTimeMs"`
}

// BatchOptions configures a batch screenshot request.
type BatchOptions struct {
	Format             string `json:"format,omitempty"`
	Width              int    `json:"width,omitempty"`
	Height             int    `json:"height,omitempty"`
	FullPage           *bool  `json:"full_page,omitempty"`
	Quality            int    `json:"quality,omitempty"`
	Delay              int    `json:"delay,omitempty"`
	DarkMode           *bool  `json:"dark_mode,omitempty"`
	BlockAds           *bool  `json:"block_ads,omitempty"`
	BlockCookieBanners *bool  `json:"block_cookie_banners,omitempty"`
	Device             string `json:"device,omitempty"`
	HideSelectors      string `json:"hide_selectors,omitempty"`
	ClickSelector      string `json:"click_selector,omitempty"`
	UserAgent          string `json:"user_agent,omitempty"`
}

// BatchJobItem represents a single URL result in a batch job.
type BatchJobItem struct {
	URL         string `json:"url"`
	Status      string `json:"status"`
	DownloadURL string `json:"downloadUrl,omitempty"`
	Error       string `json:"error,omitempty"`
}

// BatchJobResponse is the response from the batch screenshot endpoints.
type BatchJobResponse struct {
	JobID       string         `json:"jobId"`
	Status      string         `json:"status"`
	StatusURL   string         `json:"statusUrl"`
	Total       int            `json:"total"`
	Completed   int            `json:"completed"`
	Failed      int            `json:"failed"`
	Items       []BatchJobItem `json:"items"`
	CreatedAt   string         `json:"createdAt"`
	CompletedAt string         `json:"completedAt,omitempty"`
}

// WebhookCreateOptions configures a webhook creation request.
type WebhookCreateOptions struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

// WebhookResponse is a webhook returned by the API.
type WebhookResponse struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	Secret    string   `json:"secret"`
	IsActive  bool     `json:"isActive"`
	CreatedAt string   `json:"createdAt"`
}

// WebhookTestResult is the response from a webhook test delivery.
type WebhookTestResult struct {
	DeliveryID  string  `json:"deliveryId"`
	StatusCode  *int    `json:"statusCode"`
	Success     bool    `json:"success"`
	DeliveredAt *string `json:"deliveredAt"`
}

// CaptureJSONResponse is the JSON response from the screenshot endpoint.
type CaptureJSONResponse struct {
	URL              *string `json:"url"`
	Source           string  `json:"source,omitempty"`
	Format           string  `json:"format"`
	Width            int     `json:"width"`
	Height           int     `json:"height"`
	Image            string  `json:"image"`
	Size             int     `json:"size"`
	Cache            string  `json:"cache"`
	ResponseTime     string  `json:"responseTime"`
	RemainingCredits int     `json:"remainingCredits"`
}

// SignedURLResponse is the response from the sign endpoint.
type SignedURLResponse struct {
	SignedURL string `json:"signed_url"`
	ExpiresAt string `json:"expires_at"`
	ExpiresIn int    `json:"expires_in"`
}

// UsageResponse is the response from the usage endpoint.
type UsageResponse struct {
	Plan   string `json:"plan"`
	Period struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"period"`
	Usage struct {
		ScreenshotsUsed      int `json:"screenshots_used"`
		ScreenshotsLimit     int `json:"screenshots_limit"`
		ScreenshotsRemaining int `json:"screenshots_remaining"`
	} `json:"usage"`
}

// InfoResponse is the response from the screenshot info endpoint.
type InfoResponse struct {
	URL         string `json:"url"`
	Cached      bool   `json:"cached"`
	CacheKey    string `json:"cache_key"`
	CachedAt    string `json:"cached_at,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// DailyCount represents usage for a single day.
type DailyCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// UsageDailyResponse is the response from the daily usage endpoint.
type UsageDailyResponse struct {
	Days int          `json:"days"`
	Data []DailyCount `json:"data"`
}
