package anthropic

// anthropic_test.go — contract + unit tests backed by a fake local HTTP server.
// All tests here run in CI with NO network (fixtures only; ANTHROPIC_LIVE tests
// are in live_test.go and require ANTHROPIC_LIVE=1 + ANTHROPIC_API_KEY env vars).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/intelligence/porttest"
)

// ── fake server helpers ───────────────────────────────────────────────────────

// fakeServer is a recording/replay HTTP server backed by testdata fixtures.
type fakeHandler struct {
	responseFile string // path to the JSON response fixture
	statusCode   int    // HTTP status to return (default 200)
	// capturedBody stores the last request body for assertion.
	capturedBody []byte
}

func (h *fakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Capture request body.
	buf := make([]byte, 0, 512)
	tmp := make([]byte, 512)
	for {
		n, err := r.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	h.capturedBody = buf

	code := h.statusCode
	if code == 0 {
		code = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	data, err := os.ReadFile(h.responseFile)
	if err != nil {
		_, _ = w.Write([]byte(`{"error":{"type":"fixture_error","message":"` + err.Error() + `"}}`))
		return
	}
	_, _ = w.Write(data)
}

// newFakeAdapter creates an Adapter backed by a fake HTTP server that replays
// the given fixture file. Returns the adapter and a cleanup function.
func newFakeAdapter(t *testing.T, fixtureFile string, statusCode int) (*Adapter, *fakeHandler) {
	t.Helper()
	h := &fakeHandler{responseFile: fixtureFile, statusCode: statusCode}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	a := New(Config{
		APIKey:      "sk-ant-test-key-1234",
		BaseURL:     srv.URL,
		MaxAttempts: 1, // single attempt for most fixture tests
	})
	return a, h
}

// ── porttest contract suite (against fake server) ────────────────────────────

func TestAnthropicContract(t *testing.T) {
	porttest.Run(t, func(t *testing.T) core.IntelligencePort {
		t.Helper()
		a, _ := newFakeAdapter(t, "testdata/draft_response.json", 200)
		return a
	})
}

// TestModelConstants pins modelQuality/modelFast to their expected strings so a
// silent drift (e.g. a stale "-4-5" suffix left behind by a model-tier bump)
// fails loudly instead of shipping unnoticed.
func TestModelConstants(t *testing.T) {
	if modelQuality != "claude-sonnet-5" {
		t.Errorf("modelQuality = %q, want %q", modelQuality, "claude-sonnet-5")
	}
	if modelFast != "claude-haiku-4-5" {
		t.Errorf("modelFast = %q, want %q", modelFast, "claude-haiku-4-5")
	}
}

// ── capability + identity tests ───────────────────────────────────────────────

func TestCapabilities_ExcludesEmbed(t *testing.T) {
	a := New(Config{APIKey: "sk-ant-test-key-1234"})
	for _, cap := range a.Capabilities() {
		if cap.Name == "embed" {
			t.Fatalf("Capabilities: must NOT include 'embed' (CONTRA-3), got %v", a.Capabilities())
		}
	}
}

func TestCapabilities_IncludesDraftSynthesizeClassify(t *testing.T) {
	a := New(Config{APIKey: "sk-ant-test-key-1234"})
	required := []string{"draft", "synthesize", "classify"}
	caps := a.Capabilities()
	for _, name := range required {
		found := false
		for _, c := range caps {
			if c.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Capabilities: want %q in set %v", name, caps)
		}
	}
}

func TestIsAvailable_WithKey(t *testing.T) {
	a := New(Config{APIKey: "sk-ant-test-key-1234"})
	if !a.IsAvailable() {
		t.Fatal("IsAvailable: want true when key set")
	}
}

func TestIsAvailable_WithoutKey(t *testing.T) {
	a := New(Config{APIKey: ""})
	if a.IsAvailable() {
		t.Fatal("IsAvailable: want false when no key")
	}
}

func TestProviderName(t *testing.T) {
	a := New(Config{APIKey: "sk-ant-test-key-1234"})
	if a.ProviderName() != "anthropic" {
		t.Fatalf("ProviderName: want %q, got %q", "anthropic", a.ProviderName())
	}
}

// ── Draft tests ───────────────────────────────────────────────────────────────

func TestDraft_JSONOutput(t *testing.T) {
	a, _ := newFakeAdapter(t, "testdata/draft_response.json", 200)
	resp, err := a.Draft(context.Background(), core.DraftRequest{Context: "contract test"})
	if err != nil {
		t.Fatalf("Draft: unexpected error: %v", err)
	}
	if resp.Output["result"] != "draft output" {
		t.Fatalf("Draft.Output: want {result: draft output}, got %v", resp.Output)
	}
	if resp.Usage.Adapter != "anthropic" {
		t.Fatalf("Draft.Usage.Adapter: want %q, got %q", "anthropic", resp.Usage.Adapter)
	}
	if resp.Usage.TokensUsed != 18 { // 10 input + 8 output = 18
		t.Fatalf("Draft.Usage.TokensUsed: want 18, got %d", resp.Usage.TokensUsed)
	}
}

func TestDraft_UsageAdapterEqualsProviderName(t *testing.T) {
	a, _ := newFakeAdapter(t, "testdata/draft_response.json", 200)
	resp, err := a.Draft(context.Background(), core.DraftRequest{Context: "test"})
	if err != nil {
		t.Fatalf("Draft: unexpected error: %v", err)
	}
	if resp.Usage.Adapter != a.ProviderName() {
		t.Fatalf("Usage.Adapter %q != ProviderName %q", resp.Usage.Adapter, a.ProviderName())
	}
}

// ── Synthesize tests ──────────────────────────────────────────────────────────

func TestSynthesize_ReturnsText(t *testing.T) {
	a, _ := newFakeAdapter(t, "testdata/synthesize_response.json", 200)
	resp, err := a.Synthesize(context.Background(), core.SynthesisRequest{
		Query:   "contract test",
		Entries: []any{"entry1"},
		MaxLen:  100,
	})
	if err != nil {
		t.Fatalf("Synthesize: unexpected error: %v", err)
	}
	if resp.Text != "The synthesized answer." {
		t.Fatalf("Synthesize.Text: want %q, got %q", "The synthesized answer.", resp.Text)
	}
	if resp.Usage.Adapter != "anthropic" {
		t.Fatalf("Synthesize.Usage.Adapter: want %q, got %q", "anthropic", resp.Usage.Adapter)
	}
	if resp.Usage.TokensUsed != 25 { // 20 + 5
		t.Fatalf("Synthesize.Usage.TokensUsed: want 25, got %d", resp.Usage.TokensUsed)
	}
}

// ── Embed tests ───────────────────────────────────────────────────────────────

func TestEmbed_ReturnsUnavailableError(t *testing.T) {
	a := New(Config{APIKey: "sk-ant-test-key-1234"})
	_, err := a.Embed(context.Background(), "test")
	if err == nil {
		t.Fatal("Embed: want error, got nil")
	}
	if !strings.Contains(err.Error(), "CONTRA-3") {
		t.Fatalf("Embed error: want CONTRA-3 mention, got %q", err.Error())
	}
}

// ── Classify tests ────────────────────────────────────────────────────────────

func TestClassify_EmptyCategories(t *testing.T) {
	a := New(Config{APIKey: "sk-ant-test-key-1234"})
	_, err := a.Classify(context.Background(), "text", []string{})
	if err == nil {
		t.Fatal("Classify with empty categories: want error, got nil")
	}
}

func TestClassify_ReturnsCategory(t *testing.T) {
	// Serve a response where the model returns "positive".
	h := &fakeHandler{
		responseFile: "", // we'll use inline response
		statusCode:   200,
	}
	_ = h
	// Use a custom handler that returns inline JSON.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"id": "msg_classify",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "positive"}],
			"model": "claude-haiku-4-5",
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 15, "output_tokens": 1}
		}`))
	}))
	defer srv.Close()

	a := New(Config{
		APIKey:      "sk-ant-test-key-1234",
		BaseURL:     srv.URL,
		MaxAttempts: 1,
	})
	result, err := a.Classify(context.Background(), "great product", []string{"positive", "negative", "neutral"})
	if err != nil {
		t.Fatalf("Classify: unexpected error: %v", err)
	}
	if result.Category != "positive" {
		t.Fatalf("Classify.Category: want %q, got %q", "positive", result.Category)
	}
	if result.Confidence < 0 || result.Confidence > 1 {
		t.Fatalf("Classify.Confidence %v not in [0,1]", result.Confidence)
	}
}

// ── NFR-S-01: key masking / no key leakage ───────────────────────────────────

func TestMask_RedactsKey(t *testing.T) {
	cases := []struct {
		key  string
		want string
	}{
		{"", "<empty>"},
		{"abcd", "sk-ant-…abcd"},
		{"sk-ant-api03-abcdefghijklmnop-LAST", "sk-ant-…LAST"},
	}
	for _, tc := range cases {
		got := mask(tc.key)
		if got != tc.want {
			t.Errorf("mask(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestNoKeyLeakageInError(t *testing.T) {
	// Point adapter at a 401 server; the error must not contain the raw API key.
	apiKey := "sk-ant-test-secret-key-shouldnotleak-9876"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid key"}}`))
	}))
	defer srv.Close()

	a := New(Config{
		APIKey:      apiKey,
		BaseURL:     srv.URL,
		MaxAttempts: 1,
	})
	_, err := a.Draft(context.Background(), core.DraftRequest{Context: "test"})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	errStr := err.Error()
	if strings.Contains(errStr, apiKey) {
		t.Fatalf("NFR-S-01 VIOLATION: raw API key appears in error string: %q", errStr)
	}
}

// ── Usage side-channel (FR-IL-09) via fake server ─────────────────────────────

func TestUsageSideChannel_DraftReturnsUsage(t *testing.T) {
	a, _ := newFakeAdapter(t, "testdata/draft_response.json", 200)
	resp, err := a.Draft(context.Background(), core.DraftRequest{Context: "usage test"})
	if err != nil {
		t.Fatalf("Draft: %v", err)
	}
	u := resp.Usage
	if u.Adapter != "anthropic" {
		t.Errorf("Usage.Adapter: want %q, got %q", "anthropic", u.Adapter)
	}
	if u.Model == "" {
		t.Error("Usage.Model: want non-empty, got empty")
	}
	if u.TokensUsed <= 0 {
		t.Errorf("Usage.TokensUsed: want >0, got %d", u.TokensUsed)
	}
}

func TestUsageSideChannel_SynthesizeReturnsUsage(t *testing.T) {
	a, _ := newFakeAdapter(t, "testdata/synthesize_response.json", 200)
	resp, err := a.Synthesize(context.Background(), core.SynthesisRequest{
		Query:   "q",
		Entries: []any{"e"},
		MaxLen:  50,
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	u := resp.Usage
	if u.Adapter != "anthropic" {
		t.Errorf("Usage.Adapter: want %q, got %q", "anthropic", u.Adapter)
	}
	if u.TokensUsed <= 0 {
		t.Errorf("Usage.TokensUsed: want >0, got %d", u.TokensUsed)
	}
}

// ── Request wire format assertions ────────────────────────────────────────────

func TestDraft_RequestContainsAnthropicVersionHeader(t *testing.T) {
	var capturedHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("anthropic-version")
		data, _ := os.ReadFile("testdata/draft_response.json")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	a := New(Config{APIKey: "sk-ant-test-1234", BaseURL: srv.URL, MaxAttempts: 1})
	_, _ = a.Draft(context.Background(), core.DraftRequest{Context: "hdr test"})
	if capturedHeader != anthropicVersion {
		t.Fatalf("anthropic-version header: want %q, got %q", anthropicVersion, capturedHeader)
	}
}

func TestDraft_RequestBodyIsValidJSON(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		capturedBody = buf[:n]
		data, _ := os.ReadFile("testdata/draft_response.json")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	a := New(Config{APIKey: "sk-ant-test-1234", BaseURL: srv.URL, MaxAttempts: 1})
	_, _ = a.Draft(context.Background(), core.DraftRequest{Context: "body test"})
	var parsed messagesRequest
	if err := json.Unmarshal(capturedBody, &parsed); err != nil {
		t.Fatalf("request body is not valid JSON: %v\nbody: %s", err, capturedBody)
	}
	if parsed.Model != modelQuality {
		t.Fatalf("request model: want %q, got %q", modelQuality, parsed.Model)
	}
}
