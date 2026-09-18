package anthropic

// retry_test.go — unit tests for CloudRetryPolicy (retryPolicy + retryableHTTPError).

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// ── retryPolicy.Do ────────────────────────────────────────────────────────────

func TestRetryPolicy_SuccessOnFirstAttempt(t *testing.T) {
	p := newRetryPolicy(3)
	calls := 0
	err := p.Do(context.Background(), func(_ context.Context) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("Do: want nil, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls: want 1, got %d", calls)
	}
}

func TestRetryPolicy_RetriesOn429(t *testing.T) {
	p := newRetryPolicy(3)
	p.sleep = func(_ context.Context, _ time.Duration) error { return nil } // fast
	calls := 0
	err := p.Do(context.Background(), func(_ context.Context) error {
		calls++
		if calls < 3 {
			return &retryableHTTPError{statusCode: 429, message: "rate limit"}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do: want nil after retries, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls: want 3, got %d", calls)
	}
}

func TestRetryPolicy_RetriesOn503(t *testing.T) {
	p := newRetryPolicy(3)
	p.sleep = func(_ context.Context, _ time.Duration) error { return nil }
	calls := 0
	err := p.Do(context.Background(), func(_ context.Context) error {
		calls++
		if calls < 2 {
			return &retryableHTTPError{statusCode: 503, message: "service unavailable"}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do: want nil after retry, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls: want 2, got %d", calls)
	}
}

func TestRetryPolicy_NoRetryOn401(t *testing.T) {
	p := newRetryPolicy(3)
	calls := 0
	err := p.Do(context.Background(), func(_ context.Context) error {
		calls++
		return &retryableHTTPError{statusCode: 401, message: "unauthorized"}
	})
	// Must return immediately as ProviderError; only 1 attempt.
	if calls != 1 {
		t.Fatalf("calls: want 1 (no retry on 401), got %d", calls)
	}
	var pe *ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("want *ProviderError, got %T: %v", err, err)
	}
	if pe.StatusCode != 401 {
		t.Fatalf("ProviderError.StatusCode: want 401, got %d", pe.StatusCode)
	}
}

func TestRetryPolicy_NoRetryOn400(t *testing.T) {
	p := newRetryPolicy(3)
	calls := 0
	err := p.Do(context.Background(), func(_ context.Context) error {
		calls++
		return &retryableHTTPError{statusCode: 400, message: "bad request"}
	})
	if calls != 1 {
		t.Fatalf("calls: want 1 (no retry on 400), got %d", calls)
	}
	var pe *ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("want *ProviderError, got %T: %v", err, err)
	}
}

func TestRetryPolicy_ExhaustMaxAttempts(t *testing.T) {
	p := newRetryPolicy(3)
	p.sleep = func(_ context.Context, _ time.Duration) error { return nil }
	calls := 0
	err := p.Do(context.Background(), func(_ context.Context) error {
		calls++
		return &retryableHTTPError{statusCode: 500, message: "internal error"}
	})
	if calls != 3 {
		t.Fatalf("calls: want 3 (max attempts), got %d", calls)
	}
	if err == nil {
		t.Fatal("Do: want error after exhausting attempts, got nil")
	}
}

func TestRetryPolicy_CtxCancellation(t *testing.T) {
	p := newRetryPolicy(3)
	// Real sleep so cancellation is observable.
	p.sleep = ctxSleep
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately from a goroutine after first attempt.
	var calls int32
	err := p.Do(ctx, func(_ context.Context) error {
		atomic.AddInt32(&calls, 1)
		cancel() // cancel after first call
		return &retryableHTTPError{statusCode: 429, message: "rate limit"}
	})
	if err == nil {
		t.Fatal("Do: want error after ctx cancel, got nil")
	}
	// Should be context.Canceled.
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestRetryPolicy_HonoursRetryAfter(t *testing.T) {
	p := newRetryPolicy(3)
	var sleptFor time.Duration
	p.sleep = func(_ context.Context, d time.Duration) error {
		sleptFor = d
		return nil
	}
	calls := 0
	err := p.Do(context.Background(), func(_ context.Context) error {
		calls++
		if calls == 1 {
			return &retryableHTTPError{
				statusCode: 429,
				message:    "rate limit",
				retryAfter: 5 * time.Second,
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do: want nil, got %v", err)
	}
	if sleptFor != 5*time.Second {
		t.Fatalf("sleep duration: want 5s (Retry-After), got %v", sleptFor)
	}
}

// ── parseRetryAfter ───────────────────────────────────────────────────────────

func TestParseRetryAfter_Seconds(t *testing.T) {
	d := parseRetryAfter("3")
	if d != 3*time.Second {
		t.Fatalf("parseRetryAfter(\"3\"): want 3s, got %v", d)
	}
}

func TestParseRetryAfter_Empty(t *testing.T) {
	d := parseRetryAfter("")
	if d != 0 {
		t.Fatalf("parseRetryAfter(\"\"): want 0, got %v", d)
	}
}

func TestParseRetryAfter_Invalid(t *testing.T) {
	d := parseRetryAfter("not-a-date")
	if d != 0 {
		t.Fatalf("parseRetryAfter(\"not-a-date\"): want 0, got %v", d)
	}
}

// ── End-to-end retry through fake HTTP server ─────────────────────────────────

func TestAdapter_RetryOn429ThenSuccess(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(429)
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"rate limited"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"id": "msg_retry",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "ok"}],
			"model": "claude-sonnet-5",
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 5, "output_tokens": 1}
		}`))
	}))
	defer srv.Close()

	a := New(Config{
		APIKey:      "sk-ant-test-1234",
		BaseURL:     srv.URL,
		MaxAttempts: 3,
	})
	// Override sleep to be instant in tests.
	a.retry.sleep = func(_ context.Context, _ time.Duration) error { return nil }

	resp, err := a.Draft(context.Background(), core.DraftRequest{Context: "retry test"})
	if err != nil {
		t.Fatalf("Draft: want nil after retry, got %v", err)
	}
	if resp.Usage.TokensUsed != 6 {
		t.Fatalf("Usage.TokensUsed: want 6, got %d", resp.Usage.TokensUsed)
	}
	if calls != 2 {
		t.Fatalf("server calls: want 2, got %d", calls)
	}
}

func TestAdapter_ProviderErrorOn401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid key"}}`))
	}))
	defer srv.Close()

	a := New(Config{
		APIKey:      "sk-ant-test-1234",
		BaseURL:     srv.URL,
		MaxAttempts: 3,
	})
	_, err := a.Draft(context.Background(), core.DraftRequest{Context: "auth test"})
	if err == nil {
		t.Fatal("Draft: want error on 401, got nil")
	}
	var pe *ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("want *ProviderError, got %T: %v", err, err)
	}
	if pe.StatusCode != 401 {
		t.Fatalf("ProviderError.StatusCode: want 401, got %d", pe.StatusCode)
	}
}
