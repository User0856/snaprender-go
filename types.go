package snaprender

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
