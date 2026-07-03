package expr

import (
	"errors"
	"testing"

	"github.com/awis/awis/internal/expr/corpus"
)

// TestTemplateCorpusConformance asserts ParseTemplate agrees with every row of
// the frozen oracle (docs/EXPRESSION_GRAMMARS.md Table 1). The corpus is the
// oracle: if the parser disagrees, the parser is wrong. Every invalid row must
// be rejected with a *ParseError carrying an in-bounds Position.
func TestTemplateCorpusConformance(t *testing.T) {
	for _, f := range corpus.Template {
		_, err := ParseTemplate(f.Src)
		gotValid := err == nil
		if gotValid != f.Valid {
			t.Errorf("ParseTemplate(%q): got valid=%v, want valid=%v (rule %q); err=%v",
				f.Src, gotValid, f.Valid, f.Rule, err)
			continue
		}
		if !f.Valid {
			assertParseErrorInBounds(t, "ParseTemplate", f.Src, err)
		}
	}
}

// TestConditionCorpusConformance asserts ParseCondition agrees with every row of
// the frozen oracle (docs/EXPRESSION_GRAMMARS.md Table 2).
func TestConditionCorpusConformance(t *testing.T) {
	for _, f := range corpus.Condition {
		_, err := ParseCondition(f.Src)
		gotValid := err == nil
		if gotValid != f.Valid {
			t.Errorf("ParseCondition(%q): got valid=%v, want valid=%v (rule %q); err=%v",
				f.Src, gotValid, f.Valid, f.Rule, err)
			continue
		}
		if !f.Valid {
			assertParseErrorInBounds(t, "ParseCondition", f.Src, err)
		}
	}
}

// TestCorpusRowCounts pins the oracle size (42 total) so an accidental corpus
// edit is caught here as well as by the conformance loops.
func TestCorpusRowCounts(t *testing.T) {
	if got := len(corpus.Template); got != 15 {
		t.Errorf("corpus.Template rows = %d, want 15", got)
	}
	if got := len(corpus.Condition); got != 27 {
		t.Errorf("corpus.Condition rows = %d, want 27", got)
	}
}

func assertParseErrorInBounds(t *testing.T, fn, src string, err error) {
	t.Helper()
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Errorf("%s(%q): error is not *ParseError: %T (%v)", fn, src, err, err)
		return
	}
	if pe.Position < 0 || pe.Position > len(src) {
		t.Errorf("%s(%q): Position %d out of bounds [0,%d]", fn, src, pe.Position, len(src))
	}
	if pe.Msg == "" {
		t.Errorf("%s(%q): ParseError has empty Msg", fn, src)
	}
}
