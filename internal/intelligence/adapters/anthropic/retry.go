// Package anthropic — retry.go implements the CloudRetryPolicy for the Anthropic
// adapter (M16; IMP §27.M16).
//
// Policy: 429 and 5xx are retryable, honour Retry-After header (populated via
// retryableHTTPError.retryAfter), exponential backoff with jitter-free base,
// maximum 3 attempts (configurable via Config.MaxAttempts), ctx-aware.
// Non-retryable 4xx → ProviderError (typed, not retried).
package anthropic

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// retryableHTTPError carries enough information for the retry policy to decide
// whether to retry and how long to wait.
type retryableHTTPError struct {
	statusCode int
	message    string
	retryAfter time.Duration // populated from Retry-After header when > 0
}

func (e *retryableHTTPError) Error() string {
	return fmt.Sprintf("anthropic: http %d: %s", e.statusCode, e.message)
}

// isRetryable reports whether this HTTP status warrants a retry attempt.
func (e *retryableHTTPError) isRetryable() bool {
	return e.statusCode == 429 || e.statusCode >= 500
}

// asProviderError converts a non-retryable HTTP error to a *ProviderError.
func (e *retryableHTTPError) asProviderError() *ProviderError {
	return &ProviderError{StatusCode: e.statusCode, Message: e.message}
}

// parseRetryAfter converts a Retry-After header value (seconds as integer or
// HTTP-date string) to a time.Duration. Returns 0 if the header is absent or
// unparseable — the retryPolicy falls back to exponential backoff.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	secs, err := strconv.Atoi(value)
	if err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	// HTTP-date fallback: parse RFC1123 / RFC850 / asctime.
	for _, layout := range []string{
		time.RFC1123,
		time.RFC850,
		"Monday, 02-Jan-06 15:04:05 MST",
	} {
		if t, err := time.Parse(layout, value); err == nil {
			d := time.Until(t)
			if d < 0 {
				return 0
			}
			return d
		}
	}
	return 0
}

// retryPolicy holds the retry configuration for the Anthropic adapter.
type retryPolicy struct {
	maxAttempts int
	baseDelay   time.Duration
	// sleep is injectable for deterministic testing.
	sleep func(context.Context, time.Duration) error
}

// newRetryPolicy returns a retryPolicy with exponential backoff starting at
// 500 ms, up to maxAttempts attempts.
func newRetryPolicy(maxAttempts int) *retryPolicy {
	return &retryPolicy{
		maxAttempts: maxAttempts,
		baseDelay:   500 * time.Millisecond,
		sleep:       ctxSleep,
	}
}

// ctxSleep sleeps for d or until ctx is done, whichever comes first. It
// returns ctx.Err() if the context was cancelled during the sleep.
func ctxSleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Do runs fn up to maxAttempts times, retrying on retryable errors and sleeping
// between attempts. Non-retryable errors are returned immediately. If all
// attempts are exhausted the last error is returned.
func (p *retryPolicy) Do(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt < p.maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		// Inspect the error: only retryableHTTPError governs retry logic.
		var httpErr *retryableHTTPError
		isHTTPErr := isRetryableHTTPError(lastErr, &httpErr)

		if isHTTPErr && !httpErr.isRetryable() {
			// Non-retryable 4xx → typed ProviderError.
			return httpErr.asProviderError()
		}

		// Not a retriable HTTP error and not a retryableHTTPError at all
		// (e.g. network error) — still retry up to maxAttempts.
		if attempt == p.maxAttempts-1 {
			break
		}

		// Determine sleep duration: Retry-After first, then exponential backoff.
		delay := p.baseDelay * (1 << uint(attempt)) // 500ms, 1s, 2s, ...
		if isHTTPErr && httpErr.retryAfter > 0 {
			delay = httpErr.retryAfter
		}

		if err := p.sleep(ctx, delay); err != nil {
			return err
		}
	}
	return lastErr
}

// isRetryableHTTPError checks whether err is a *retryableHTTPError (directly or
// wrapped). If so it sets *out and returns true.
func isRetryableHTTPError(err error, out **retryableHTTPError) bool {
	if err == nil {
		return false
	}
	if he, ok := err.(*retryableHTTPError); ok {
		*out = he
		return true
	}
	return false
}
