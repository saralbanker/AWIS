package expr

import (
	"testing"

	"github.com/awis/awis/internal/expr/corpus"
)

// nasties are pathological inputs seeded into both fuzzers alongside the corpus.
var nasties = []string{
	"", "{{", "}}", "{{}}", "{{ }}", "{{{{}}}}", "{{a", "a}}", "{",
	"''", "'", "(((", ")))", "()", "!", "&&", "||", "==", "> null",
	"null == null", "steps..status", "steps.-.status", "event.",
	"{{workflow.inputs." + longIdent() + "}}",
	"event.count == " + longDigits(),
}

func longIdent() string {
	b := make([]byte, 4096)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

func longDigits() string {
	b := make([]byte, 300)
	for i := range b {
		b[i] = '9'
	}
	return string(b)
}

// FuzzParseTemplate: no input may panic ParseTemplate, and any accepted template
// must Resolve against an empty Env without panicking.
func FuzzParseTemplate(f *testing.F) {
	for _, fx := range corpus.Template {
		f.Add(fx.Src)
	}
	for _, s := range nasties {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		tmpl, err := ParseTemplate(src)
		if err != nil {
			if pe, ok := err.(*ParseError); ok {
				if pe.Position < 0 || pe.Position > len(src) {
					t.Fatalf("ParseTemplate(%q): Position %d out of bounds [0,%d]", src, pe.Position, len(src))
				}
			}
			return
		}
		// Must not panic; result is ignored.
		_, _ = tmpl.Resolve(Env{})
	})
}

// FuzzParseCondition: no input may panic ParseCondition, and any accepted
// condition must Eval against an empty Env without panicking.
func FuzzParseCondition(f *testing.F) {
	for _, fx := range corpus.Condition {
		f.Add(fx.Src)
	}
	for _, s := range nasties {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		ce, err := ParseCondition(src)
		if err != nil {
			if pe, ok := err.(*ParseError); ok {
				if pe.Position < 0 || pe.Position > len(src) {
					t.Fatalf("ParseCondition(%q): Position %d out of bounds [0,%d]", src, pe.Position, len(src))
				}
			}
			return
		}
		// Must not panic; result is ignored.
		_, _ = ce.Eval(Env{})
	})
}
