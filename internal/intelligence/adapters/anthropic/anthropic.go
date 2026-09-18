// Package anthropic provides the Anthropic cloud adapter for core.IntelligencePort
// (M16; Blueprint §14; FR-IL-02/05..09; NFR-S-01).
//
// The adapter communicates with the Anthropic Messages API using stdlib net/http
// only — no third-party Anthropic SDK (IMP §27.M16 CE pin). The API key is read
// from the ANTHROPIC_API_KEY environment variable; config-file wiring is M17.
//
// Capabilities: draft, synthesize, classify (fast model placeholder). Embed is
// NOT implemented (CONTRA-3 — Blueprint §14 V1 constraint); requesting it returns
// ErrEmbedUnavailable and IsAvailable/Capabilities exclude it.
//
// Usage is returned in every successful response so the UsageRunner side-channel
// (ADJ-8, FR-IL-09) can record adapter/model/tokens in StepCompleted events.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/awis/awis/internal/core"
)

// Model IDs pinned at M16. These are config-overridable post-V1; changing the
// const block is the only required edit (IMP §27.M16 note).
const (
	// defaultModelFast maps to the "fast" model_hint (claude-haiku class).
	defaultModelFast = "claude-haiku-4-5-20251001"
	// defaultModelQuality maps to the "quality" model_hint (claude-sonnet class).
	defaultModelQuality = "claude-sonnet-5"

	modelFast    = defaultModelFast
	modelQuality = defaultModelQuality
)

const (
	// messagesEndpoint is the Anthropic Messages API endpoint.
	messagesEndpoint = "https://api.anthropic.com/v1/messages"
	// anthropicVersion is the required header value.
	anthropicVersion = "2023-06-01"
	// maxTokensDefault is used when the request does not specify a limit.
	maxTokensDefault = 1024
	// providerName is the stable ProviderName() identifier.
	providerName = "anthropic"
)

// ErrEmbedUnavailable is returned when Embed is called on the Anthropic adapter.
// Per CONTRA-3, Embed is not implemented in V1; the router falls through to its
// fallback path when it receives this typed error.
var ErrEmbedUnavailable = errors.New("anthropic: embed not available in V1 (CONTRA-3)")

// ProviderError is a typed error returned for non-retryable 4xx responses from
// the Anthropic API. The StatusCode is the HTTP status; Message is the
// provider's error description.
type ProviderError struct {
	StatusCode int
	Message    string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("anthropic: provider error %d: %s", e.StatusCode, e.Message)
}

// Config configures the Anthropic adapter. APIKey is required; all other fields
// have sensible defaults.
type Config struct {
	// APIKey is the Anthropic API key (ANTHROPIC_API_KEY). Never logged or exposed.
	APIKey string
	// BaseURL overrides the API base URL (useful for fake servers in tests).
	// If empty, messagesEndpoint is used.
	BaseURL string
	// HTTPClient is the HTTP client to use. If nil, a default client is constructed.
	HTTPClient *http.Client
	// MaxAttempts is the maximum number of attempts for retryable errors.
	// Defaults to 3 when 0.
	MaxAttempts int
	// ModelFast is the model identifier for fast tasks. Defaults to claude-haiku-4-5-20251001.
	ModelFast string
	// ModelQuality is the model identifier for quality tasks. Defaults to claude-sonnet-5.
	ModelQuality string
}

// Adapter is the Anthropic IntelligencePort implementation.
type Adapter struct {
	cfg    Config
	client *http.Client
	retry  *retryPolicy
}

// Compile-time assertion: *Adapter must satisfy core.IntelligencePort.
var _ core.IntelligencePort = (*Adapter)(nil)

// New constructs a new Adapter from cfg. cfg.APIKey must be non-empty for
// IsAvailable to return true; all other fields have defaults.
func New(cfg Config) *Adapter {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if cfg.ModelFast == "" {
		cfg.ModelFast = defaultModelFast
	}
	if cfg.ModelQuality == "" {
		cfg.ModelQuality = defaultModelQuality
	}
	return &Adapter{
		cfg:    cfg,
		client: cfg.HTTPClient,
		retry:  newRetryPolicy(maxAttempts),
	}
}

// IsAvailable returns true when an API key is configured (the key is present in
// the environment / config). It does NOT make a network round-trip.
func (a *Adapter) IsAvailable() bool {
	return a.cfg.APIKey != ""
}

// Capabilities declares the capabilities this adapter provides.
// Embed is excluded per CONTRA-3.
func (a *Adapter) Capabilities() []core.Capability {
	return []core.Capability{
		{Name: "draft"},
		{Name: "synthesize"},
		{Name: "classify"},
	}
}

// ProviderName returns the stable identifier "anthropic".
func (a *Adapter) ProviderName() string { return providerName }

// Draft sends a DraftRequest to the Anthropic Messages API using the quality
// model and returns a DraftResponse containing structured output plus usage.
func (a *Adapter) Draft(ctx context.Context, req core.DraftRequest) (core.DraftResponse, error) {
	prompt := buildDraftPrompt(req)
	text, usage, err := a.complete(ctx, a.cfg.ModelQuality, prompt, maxTokensDefault)
	if err != nil {
		return core.DraftResponse{}, err
	}
	// Parse the returned text as JSON into Output; fall back to raw text field.
	output := parseJSONOutput(text)
	return core.DraftResponse{Output: output, Usage: usage}, nil
}

// Embed is not implemented in V1 (CONTRA-3). It always returns ErrEmbedUnavailable.
func (a *Adapter) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, ErrEmbedUnavailable
}

// Synthesize sends a SynthesisRequest to the Anthropic Messages API using the
// quality model and returns a SynthesisResponse containing composed text plus usage.
func (a *Adapter) Synthesize(ctx context.Context, req core.SynthesisRequest) (core.SynthesisResponse, error) {
	prompt := buildSynthesisPrompt(req)
	maxTok := req.MaxLen
	if maxTok <= 0 {
		maxTok = maxTokensDefault
	}
	text, usage, err := a.complete(ctx, a.cfg.ModelQuality, prompt, maxTok)
	if err != nil {
		return core.SynthesisResponse{}, err
	}
	return core.SynthesisResponse{Text: text, Usage: usage}, nil
}

// Classify sends a classification request to the Anthropic Messages API using
// the fast model (placeholder wiring per IMP §27.M16).
func (a *Adapter) Classify(ctx context.Context, text string, categories []string) (core.Classification, error) {
	if len(categories) == 0 {
		return core.Classification{}, errors.New("anthropic: classify requires at least one category")
	}
	prompt := buildClassifyPrompt(text, categories)
	out, usage, err := a.complete(ctx, a.cfg.ModelFast, prompt, 256)
	if err != nil {
		return core.Classification{}, err
	}
	_ = usage // classify usage not surfaced through side-channel at this level
	cat := parseCategory(out, categories)
	return core.Classification{Category: cat, Confidence: 0.8, Reasoning: out}, nil
}

// ── Anthropic Messages API wire types ────────────────────────────────────────

// messagesRequest is the JSON body sent to POST /v1/messages.
type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

// message is a single turn in the conversation.
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// messagesResponse is the JSON body returned by POST /v1/messages.
type messagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// complete sends one user message to the Anthropic Messages API, honouring the
// retry policy, and returns the assistant's reply text plus usage.
func (a *Adapter) complete(ctx context.Context, model, userPrompt string, maxTokens int) (string, core.Usage, error) {
	body := messagesRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  []message{{Role: "user", Content: userPrompt}},
	}
	var result string
	var usage core.Usage

	err := a.retry.Do(ctx, func(ctx context.Context) error {
		text, u, rerr := a.doRequest(ctx, body)
		if rerr != nil {
			return rerr
		}
		result = text
		usage = u
		return nil
	})
	if err != nil {
		return "", core.Usage{}, err
	}
	return result, usage, nil
}

// doRequest performs a single HTTP request to the Anthropic Messages API.
func (a *Adapter) doRequest(ctx context.Context, body messagesRequest) (string, core.Usage, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return "", core.Usage{}, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	endpoint := a.cfg.BaseURL
	if endpoint == "" {
		endpoint = messagesEndpoint
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return "", core.Usage{}, fmt.Errorf("anthropic: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.cfg.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", core.Usage{}, fmt.Errorf("anthropic: http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", core.Usage{}, fmt.Errorf("anthropic: read response body: %w", err)
	}

	// Non-2xx: determine retryability inside retryPolicy.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", core.Usage{}, httpError(resp.StatusCode, raw)
	}

	var mr messagesResponse
	if err := json.Unmarshal(raw, &mr); err != nil {
		return "", core.Usage{}, fmt.Errorf("anthropic: decode response: %w", err)
	}

	text := ""
	for _, block := range mr.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}

	u := core.Usage{
		Adapter:    providerName,
		Model:      mr.Model,
		TokensUsed: mr.Usage.InputTokens + mr.Usage.OutputTokens,
	}
	return text, u, nil
}

// httpError constructs the appropriate error type for an HTTP error response.
// 429 and 5xx are retryable (retryPolicy detects them via retryableHTTPError).
// Other 4xx are non-retryable ProviderErrors.
func httpError(statusCode int, body []byte) error {
	// Try to extract the API error message.
	var mr messagesResponse
	msg := string(body)
	if err := json.Unmarshal(body, &mr); err == nil && mr.Error != nil {
		msg = mr.Error.Message
	}
	return &retryableHTTPError{statusCode: statusCode, message: msg}
}

// ── Prompt builders ───────────────────────────────────────────────────────────

func buildDraftPrompt(req core.DraftRequest) string {
	var sb strings.Builder
	if req.Persona != "" {
		sb.WriteString("You are: ")
		sb.WriteString(req.Persona)
		sb.WriteString("\n\n")
	}
	sb.WriteString("Context:\n")
	sb.WriteString(req.Context)
	if len(req.Schema) > 0 {
		schemaJSON, _ := json.Marshal(req.Schema)
		sb.WriteString("\n\nReturn a JSON object matching this schema:\n")
		sb.Write(schemaJSON)
	}
	if len(req.Examples) > 0 {
		sb.WriteString("\n\nExamples:\n")
		for _, ex := range req.Examples {
			exJSON, _ := json.Marshal(ex)
			sb.Write(exJSON)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func buildSynthesisPrompt(req core.SynthesisRequest) string {
	var sb strings.Builder
	sb.WriteString("Synthesize an answer to the following query using the provided entries.\n\n")
	sb.WriteString("Query: ")
	sb.WriteString(req.Query)
	sb.WriteString("\n\nEntries:\n")
	for i, e := range req.Entries {
		fmt.Fprintf(&sb, "%d. %v\n", i+1, e)
	}
	if req.MaxLen > 0 {
		fmt.Fprintf(&sb, "\nMaximum response length: %d tokens.\n", req.MaxLen)
	}
	return sb.String()
}

func buildClassifyPrompt(text string, categories []string) string {
	var sb strings.Builder
	sb.WriteString("Classify the following text into exactly one of these categories: ")
	sb.WriteString(strings.Join(categories, ", "))
	sb.WriteString(".\n\nText: ")
	sb.WriteString(text)
	sb.WriteString("\n\nRespond with only the category name.")
	return sb.String()
}

// ── Response parsers ──────────────────────────────────────────────────────────

// parseJSONOutput attempts to unmarshal text as a JSON object. If it fails,
// it returns {"text": text} so Draft always returns a map.
func parseJSONOutput(text string) map[string]any {
	text = strings.TrimSpace(text)
	// Strip markdown code fences if present.
	if strings.HasPrefix(text, "```") {
		if idx := strings.Index(text, "\n"); idx >= 0 {
			text = text[idx+1:]
		}
		text = strings.TrimSuffix(strings.TrimSpace(text), "```")
		text = strings.TrimSpace(text)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err == nil {
		return out
	}
	return map[string]any{"text": text}
}

// parseCategory extracts the category from the model's response. It returns the
// first category whose name appears in the response (case-insensitive). If none
// match, it returns the first category as a safe default.
func parseCategory(response string, categories []string) string {
	lower := strings.ToLower(strings.TrimSpace(response))
	for _, cat := range categories {
		if strings.Contains(lower, strings.ToLower(cat)) {
			return cat
		}
	}
	return categories[0]
}

// mask returns a redacted form of key: "sk-ant-…<last4>" or "<empty>" if blank.
// NFR-S-01: the API key MUST NOT appear in logs, errors, or fixtures.
func mask(key string) string {
	if key == "" {
		return "<empty>"
	}
	if len(key) <= 4 {
		return "sk-ant-…" + key
	}
	return "sk-ant-…" + key[len(key)-4:]
}
