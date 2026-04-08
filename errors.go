package snaprender

import "fmt"

// Error represents an API error returned by SnapRender.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("snaprender: %s (%d): %s", e.Code, e.Status, e.Message)
}

// IsUnauthorized returns true if the error is a 401 authentication failure.
func (e *Error) IsUnauthorized() bool { return e.Status == 401 }

// IsRateLimited returns true if the error is a 429 rate limit or quota exceeded.
func (e *Error) IsRateLimited() bool { return e.Status == 429 }

// IsNotFound returns true if the error is a 404.
func (e *Error) IsNotFound() bool { return e.Status == 404 }
