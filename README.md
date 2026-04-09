# SnapRender Go SDK

Official Go client for the [SnapRender Screenshot API](https://snap-render.com).

## Installation

```bash
go get github.com/User0856/snaprender-go
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "os"

    snaprender "github.com/User0856/snaprender-go"
)

func main() {
    client := snaprender.NewClient("sk_live_YOUR_API_KEY")

    // Capture a screenshot
    img, err := client.Capture(context.Background(), "https://example.com", nil)
    if err != nil {
        log.Fatal(err)
    }
    os.WriteFile("screenshot.png", img, 0644)
}
```

## Features

- URL, HTML, and Markdown screenshots
- Signed URLs (shareable, no API key needed)
- Content extraction (Markdown, text, HTML, article, links, metadata)
- Batch screenshots (up to 50 URLs per job)
- Webhooks for async notifications
- Device emulation (iPhone, iPad, Pixel, MacBook)
- Dark mode, ad blocking, cookie banner removal
- Full-page capture, custom viewports
- JSON response mode for AI integrations
- Cache status checks
- Usage tracking with daily breakdowns
- Zero external dependencies (stdlib only)
- Context support for cancellation and timeouts

## Client Configuration

```go
// Default client
client := snaprender.NewClient("sk_live_YOUR_API_KEY")

// Custom base URL (for self-hosted or testing)
client := snaprender.NewClient("sk_live_YOUR_API_KEY",
    snaprender.WithBaseURL("https://your-instance.example.com"),
)

// Custom HTTP client
client := snaprender.NewClient("sk_live_YOUR_API_KEY",
    snaprender.WithHTTPClient(&http.Client{
        Transport: &http.Transport{MaxIdleConns: 20},
    }),
)

// Custom timeout (default is 60s)
client := snaprender.NewClient("sk_live_YOUR_API_KEY",
    snaprender.WithTimeout(120 * time.Second),
)
```

## Examples

### Capture with options

```go
img, err := client.Capture(ctx, "https://example.com", &snaprender.CaptureOptions{
    Format:   "jpeg",
    Width:    1920,
    DarkMode: snaprender.Bool(true),
    Device:   "iphone_15_pro",
})
```

All capture options:

| Field | Type | Description |
|-------|------|-------------|
| `Format` | `string` | `"png"`, `"jpeg"`, or `"webp"` |
| `Width` | `int` | Viewport width in pixels |
| `Height` | `int` | Viewport height in pixels |
| `FullPage` | `*bool` | Capture the full scrollable page |
| `Quality` | `int` | JPEG/WebP quality (1-100) |
| `Delay` | `int` | Wait time in ms before capture |
| `DarkMode` | `*bool` | Enable dark mode |
| `BlockAds` | `*bool` | Block ads on the page |
| `BlockCookieBanners` | `*bool` | Remove cookie consent banners |
| `Device` | `string` | Device emulation preset |
| `HideSelectors` | `string` | CSS selectors to hide |
| `ClickSelector` | `string` | CSS selector to click before capture |
| `UserAgent` | `string` | Custom User-Agent string |
| `Cache` | `*bool` | Enable/disable caching |
| `CacheTTL` | `int` | Cache time-to-live in seconds |
| `ResponseType` | `string` | Set to `"json"` for JSON response |

Use `snaprender.Bool(true)` or `snaprender.Bool(false)` for `*bool` fields.

### Render HTML

```go
html := "<html><body><h1>Hello</h1></body></html>"
img, err := client.CaptureHTML(ctx, html, nil)
```

### Render Markdown

```go
img, err := client.CaptureMarkdown(ctx, "# Hello World\n\nThis is **bold**.", nil)
```

### JSON response (for AI agents)

Returns structured metadata with a base64 data URI instead of raw bytes.

```go
result, err := client.CaptureJSON(ctx, "https://example.com", nil)
fmt.Println(result.Format)   // "png"
fmt.Println(result.Width)    // 1280
fmt.Println(result.Height)   // 800
fmt.Println(result.Image)    // "data:image/png;base64,..."
fmt.Println(result.Cache)    // "HIT" or "MISS"
```

### Generate signed URL

Signed URLs let you embed screenshots in `<img>` tags, emails, or share them without exposing your API key. Signing is free and does not count against your quota.

```go
result, err := client.Sign(ctx, "https://example.com", &snaprender.SignOptions{
    ExpiresIn: 86400, // 1 day in seconds
    Format:    "jpeg",
    DarkMode:  snaprender.Bool(true),
})
fmt.Println(result.SignedURL)  // Use in <img> tags, emails, etc.
fmt.Println(result.ExpiresAt)  // ISO 8601 expiry timestamp
```

### Content extraction

Extract content from web pages in various formats: `markdown`, `text`, `html`, `article`, `links`, or `metadata`.

```go
result, err := client.Extract(ctx, "https://example.com/blog/post", &snaprender.ExtractOptions{
    Type:     "markdown",
    Selector: "article",    // optional: target a specific element
    BlockAds: snaprender.Bool(true),
})

// result.Content is json.RawMessage, unmarshal based on type
var content string
json.Unmarshal(result.Content, &content)
fmt.Println(content)
fmt.Printf("Word count: %d\n", *result.WordCount)
```

Extract options:

| Field | Type | Description |
|-------|------|-------------|
| `Type` | `string` | `"markdown"`, `"text"`, `"html"`, `"article"`, `"links"`, `"metadata"` |
| `Selector` | `string` | CSS selector to scope extraction |
| `BlockAds` | `*bool` | Block ads on the page |
| `BlockCookieBanners` | `*bool` | Remove cookie consent banners |
| `Delay` | `int` | Wait time in ms before extraction |
| `MaxLength` | `int` | Truncate content to this character count |
| `Cache` | `*bool` | Enable/disable caching |
| `CacheTTL` | `int` | Cache time-to-live in seconds |

### Batch screenshots

Submit up to 50 URLs in one request. The job processes asynchronously; poll with `GetBatchStatus` until it completes.

```go
// Submit the batch
job, err := client.Batch(ctx, []string{
    "https://example.com",
    "https://example.org",
    "https://example.net",
}, &snaprender.BatchOptions{
    Format:   "webp",
    DarkMode: snaprender.Bool(true),
})
fmt.Printf("Job %s submitted (%d URLs)\n", job.JobID, job.Total)

// Poll for completion
for {
    status, err := client.GetBatchStatus(ctx, job.JobID)
    if err != nil {
        log.Fatal(err)
    }
    if status.Status == "completed" || status.Status == "failed" {
        for _, item := range status.Items {
            fmt.Printf("%s: %s (download: %s)\n", item.URL, item.Status, item.DownloadURL)
        }
        break
    }
    time.Sleep(2 * time.Second)
}
```

Batch options support the same capture parameters: `Format`, `Width`, `Height`, `FullPage`, `Quality`, `Delay`, `DarkMode`, `BlockAds`, `BlockCookieBanners`, `Device`, `HideSelectors`, `ClickSelector`, `UserAgent`.

### Webhooks

Register webhooks to receive notifications for events like `screenshot.completed` and `batch.completed`. Maximum 5 webhooks per account.

#### Create a webhook

```go
wh, err := client.CreateWebhook(ctx, &snaprender.WebhookCreateOptions{
    URL:    "https://your-server.com/webhooks/snaprender",
    Events: []string{"screenshot.completed", "batch.completed"},
})
fmt.Printf("Webhook ID: %s\n", wh.ID)
fmt.Printf("Secret: %s\n", wh.Secret) // Save this for signature verification
```

#### List webhooks

```go
webhooks, err := client.ListWebhooks(ctx)
for _, wh := range webhooks {
    fmt.Printf("%s -> %s (active: %v)\n", wh.ID, wh.URL, wh.IsActive)
}
```

#### Test a webhook

Send a test payload to verify your endpoint is receiving deliveries.

```go
result, err := client.TestWebhook(ctx, webhookID)
fmt.Printf("Delivery %s, success: %v\n", result.DeliveryID, result.Success)
```

#### Delete a webhook

```go
err := client.DeleteWebhook(ctx, webhookID)
```

#### Verify webhook signatures

When your server receives a webhook, verify the HMAC-SHA256 signature to confirm it came from SnapRender. `VerifyWebhookSignature` is a standalone function (not a client method) since it does not require an API key.

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("X-Webhook-Signature")

    if !snaprender.VerifyWebhookSignature(string(body), signature, webhookSecret) {
        http.Error(w, "invalid signature", http.StatusForbidden)
        return
    }

    // Process the verified payload
    fmt.Println(string(body))
    w.WriteHeader(http.StatusOK)
}
```

### Check usage

```go
usage, err := client.Usage(ctx)
fmt.Printf("Plan: %s\n", usage.Plan)
fmt.Printf("Period: %s to %s\n", usage.Period.Start, usage.Period.End)
fmt.Printf("Used: %d / %d (%d remaining)\n",
    usage.Usage.ScreenshotsUsed,
    usage.Usage.ScreenshotsLimit,
    usage.Usage.ScreenshotsRemaining,
)
```

### Daily usage breakdown

Get a per-day usage breakdown for the last N days (1-90, default 30).

```go
daily, err := client.UsageDaily(ctx, 7) // last 7 days
for _, day := range daily.Data {
    fmt.Printf("%s: %d screenshots\n", day.Date, day.Count)
}
```

### Cache status check

Check whether a screenshot is cached without actually capturing it. The cache key depends on the URL and all capture parameters. This does not count against your quota.

```go
info, err := client.Info(ctx, "https://example.com", &snaprender.CaptureOptions{
    Format: "png",
    Width:  1280,
})
fmt.Printf("Cached: %v\n", info.Cached)
if info.Cached {
    fmt.Printf("Cached at: %s, expires: %s\n", info.CachedAt, info.ExpiresAt)
}
```

### Error handling

All API errors are returned as `*snaprender.Error`, which implements the `error` interface. Use `errors.As` to inspect details.

```go
img, err := client.Capture(ctx, "https://example.com", nil)
if err != nil {
    var apiErr *snaprender.Error
    if errors.As(err, &apiErr) {
        fmt.Printf("Code: %s, Status: %d, Message: %s\n", apiErr.Code, apiErr.Status, apiErr.Message)

        if apiErr.IsRateLimited() {
            // Handle quota exceeded (HTTP 429)
        }
        if apiErr.IsUnauthorized() {
            // Handle bad API key (HTTP 401)
        }
        if apiErr.IsNotFound() {
            // Handle not found (HTTP 404)
        }
    }
}
```

## API Reference

| Method | Description |
|--------|-------------|
| `Capture(ctx, url, opts)` | Screenshot a URL (returns image bytes) |
| `CaptureHTML(ctx, html, opts)` | Screenshot from raw HTML |
| `CaptureMarkdown(ctx, md, opts)` | Screenshot from Markdown |
| `CaptureJSON(ctx, url, opts)` | Screenshot returning JSON with base64 image |
| `Sign(ctx, url, opts)` | Generate a signed URL (free, no quota) |
| `Extract(ctx, url, opts)` | Extract content from a web page |
| `Batch(ctx, urls, opts)` | Submit a batch screenshot job (1-50 URLs) |
| `GetBatchStatus(ctx, jobID)` | Poll a batch job for status and results |
| `CreateWebhook(ctx, opts)` | Register a new webhook (max 5) |
| `ListWebhooks(ctx)` | List all webhooks for the account |
| `DeleteWebhook(ctx, webhookID)` | Delete a webhook by ID |
| `TestWebhook(ctx, webhookID)` | Send a test delivery to a webhook |
| `Usage(ctx)` | Get current month's usage stats |
| `UsageDaily(ctx, days)` | Get daily usage breakdown (1-90 days) |
| `Info(ctx, url, opts)` | Check cache status for a screenshot |

Standalone functions (no client needed):

| Function | Description |
|----------|-------------|
| `VerifyWebhookSignature(payload, signature, secret)` | Verify HMAC-SHA256 webhook signature |
| `Bool(v)` | Helper to create `*bool` for option fields |

## License

MIT
