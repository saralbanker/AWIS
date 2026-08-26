package main

// trace_test.go — tests for the 'awis trace' truncation helper (defect B-7).
//
// PROVEN LIVE: 'awis --json trace <id>' printed nothing and exited 0 when a
// truncated event payload contained an unescaped '"'; the hand-built
// json.RawMessage was invalid JSON, enc.Encode failed, and (defect B-8) the
// error was discarded.

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTruncatePayloadForTraceJSON(t *testing.T) {
	cases := []struct {
		name    string
		payload string // raw bytes of the event payload (as stored, pre-truncation)
	}{
		{
			name:    "short_ascii_unaffected",
			payload: `{"a":1}`,
		},
		{
			name:    "long_ascii_no_special_chars",
			payload: `{"note":"` + strings.Repeat("x", 200) + `"}`,
		},
		{
			name: "contains_unescaped_double_quote_at_truncation_boundary",
			// This is the exact shape of the live defect: a raw JSON payload
			// whose first 120 characters end mid-way through a quoted value,
			// so naive `"` + s[:120] + `"` wrapping produces invalid JSON.
			payload: `{"description":"` + strings.Repeat("a", 150) + `","other":"value with \"nested\" quotes and more padding to exceed one hundred and twenty characters total length"}`,
		},
		{
			name:    "contains_backslash_and_newline",
			payload: `{"text":"line one\nline two\\ with backslash ` + strings.Repeat("z", 150) + `"}`,
		},
		{
			name: "multibyte_runes_exceed_120_runes",
			// Each "€" is 3 bytes in UTF-8; use enough runes to exceed the
			// 120-rune truncation boundary without landing on a byte
			// boundary that would split a rune under naive byte slicing.
			payload: `{"emoji":"` + strings.Repeat("€", 130) + `"}`,
		},
		{
			name: "multibyte_runes_over_120_bytes_but_under_120_runes",
			// >120 bytes (triggers the len(payload) > 120 check in trace.go)
			// but <=120 runes, so no visual "..." truncation happens — the
			// function must still produce valid escaped JSON.
			payload: `{"x":"` + strings.Repeat("é", 60) + `"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := truncatePayloadForTraceJSON(json.RawMessage(tc.payload))
			if err != nil {
				t.Fatalf("truncatePayloadForTraceJSON: %v", err)
			}

			// The result must always be valid JSON (defect B-7 regression guard):
			// unmarshal it and confirm it decodes to a JSON string.
			var s string
			if err := json.Unmarshal(got, &s); err != nil {
				t.Fatalf("truncatePayloadForTraceJSON(%q) = %q, which is not valid JSON: %v", tc.payload, got, err)
			}

			// It must also round-trip through a full encoder the same way
			// emitJSON/enc.Encode does, proving the overall 'trace --json'
			// encode this feeds into cannot fail on this payload.
			type wrapper struct {
				Payload json.RawMessage `json:"payload"`
			}
			if _, err := json.Marshal(wrapper{Payload: got}); err != nil {
				t.Fatalf("wrapping %q in an outer struct failed to marshal: %v", got, err)
			}

			// Rune-boundary safety: the decoded string must never end in a
			// partially-cut multi-byte rune (json.Unmarshal above already
			// guarantees the bytes are valid UTF-8/JSON, but also assert the
			// visible content is a whole number of runes by round-tripping
			// rune count).
			if strings.HasSuffix(s, "...") {
				body := strings.TrimSuffix(s, "...")
				if want := 120; len([]rune(body)) != want {
					t.Errorf("truncated body has %d runes, want exactly %d", len([]rune(body)), want)
				}
			}
		})
	}
}

// TestTruncatePayloadForTraceJSONExactBoundary proves the function does not
// truncate (and appends no "...") when the payload is exactly 120 runes.
func TestTruncatePayloadForTraceJSONExactBoundary(t *testing.T) {
	body := strings.Repeat("a", 120)
	payload := json.RawMessage(body)

	got, err := truncatePayloadForTraceJSON(payload)
	if err != nil {
		t.Fatalf("truncatePayloadForTraceJSON: %v", err)
	}
	var s string
	if err := json.Unmarshal(got, &s); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if strings.Contains(s, "...") {
		t.Errorf("payload of exactly 120 runes should not be truncated further; got %q", s)
	}
}
