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
- Device emulation (iPhone, iPad, Pixel, MacBook)
- Dark mode, ad blocking, cookie banner removal
- Full-page capture, custom viewports
- JSON response mode for AI integrations
- Zero external dependencies (stdlib only)
- Context support for cancellation and timeouts

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

### Render HTML

```go
img, err := client.CaptureHTML(ctx, "<html><body><h1>Hello</h1></body></html>", nil)
```

### Render Markdown

```go
img, err := client.CaptureMarkdown(ctx, "# Hello World\n\nThis is **bold**.", nil)
```

### Generate signed URL

```go
result, err := client.Sign(ctx, "https://example.com", &snaprender.SignOptions{
    ExpiresIn: 86400, // 1 day
    Format:    "jpeg",
    DarkMode:  snaprender.Bool(true),
})
fmt.Println(result.SignedURL) // Use in <img> tags, emails, etc.
```

### JSON response (for AI agents)

```go
result, err := client.CaptureJSON(ctx, "https://example.com", nil)
fmt.Println(result.Image) // base64 data URI
```

### Check usage

```go
usage, err := client.Usage(ctx)
fmt.Printf("Used: %d / %d\n", usage.Usage.ScreenshotsUsed, usage.Usage.ScreenshotsLimit)
```

### Error handling

```go
img, err := client.Capture(ctx, "https://example.com", nil)
if err != nil {
    var apiErr *snaprender.Error
    if errors.As(err, &apiErr) {
        if apiErr.IsRateLimited() {
            // Handle quota exceeded
        }
        fmt.Printf("API error: %s (%s)\n", apiErr.Message, apiErr.Code)
    }
}
```

## API Reference

| Method | Description |
|--------|-------------|
| `Capture(ctx, url, opts)` | Screenshot a URL (returns bytes) |
| `CaptureHTML(ctx, html, opts)` | Screenshot raw HTML |
| `CaptureMarkdown(ctx, md, opts)` | Screenshot Markdown |
| `CaptureJSON(ctx, url, opts)` | Screenshot with JSON metadata |
| `Sign(ctx, url, opts)` | Generate a signed URL |
| `Usage(ctx)` | Get current usage stats |

## License

MIT
