package expr

import (
	"testing"
)

func evalCond(t *testing.T, src string, env Env) bool {
	t.Helper()
	ce, err := ParseCondition(src)
	if err != nil {
		t.Fatalf("ParseCondition(%q) unexpected error: %v", src, err)
	}
	got, err := ce.Eval(env)
	if err != nil {
		t.Fatalf("Eval(%q) unexpected error: %v", src, err)
	}
	return got
}

// TestConditionNullRules covers the frozen null-handling rules verbatim.
func TestConditionNullRules(t *testing.T) {
	empty := Env{} // every path resolves to null/missing

	cases := []struct {
		src  string
		want bool
	}{
		{"steps.x.status == null", true},  // null == null ⇒ true
		{"steps.x.status != null", false}, // missing != null ⇒ false
		{"steps.x.status == 'done'", false},
		{"steps.x.status != 'done'", true},
		{"event.count > 5", false},  // null vs number, ordering ⇒ false
		{"event.count < 5", false},  // frozen rule
		{"event.count >= 5", false}, // frozen rule
		{"event.count <= 5", false}, // frozen rule
		{"event.count == 5", false}, // null != non-null
		{"event.count != 5", true},
	}
	for _, c := range cases {
		if got := evalCond(t, c.src, empty); got != c.want {
			t.Errorf("Eval(%q) on empty env = %v, want %v", c.src, got, c.want)
		}
	}

	// Present-and-non-null path: != null ⇒ true, == null ⇒ false.
	present := Env{StepStatus: map[string]string{"x": "completed"}}
	if got := evalCond(t, "steps.x.status != null", present); !got {
		t.Error("present status != null should be true")
	}
	if got := evalCond(t, "steps.x.status == null", present); got {
		t.Error("present status == null should be false")
	}
}

// TestConditionNumericMatrix covers EDR-010 numeric comparison for every op,
// with float64 (JSON) and Go int values.
func TestConditionNumericMatrix(t *testing.T) {
	cases := []struct {
		src  string
		val  any
		want bool
	}{
		{"event.count == 5", float64(5), true},
		{"event.count == 5", float64(6), false},
		{"event.count != 5", float64(6), true},
		{"event.count > 5", float64(6), true},
		{"event.count > 5", float64(5), false},
		{"event.count < 5", float64(4), true},
		{"event.count >= 5", float64(5), true},
		{"event.count <= 5", float64(5), true},
		{"event.count == 5", int(5), true},  // Go int accepted
		{"event.count > 5", int64(6), true}, // Go int64 accepted
		{"event.score == 3.14", 3.14, true}, // decimal
		{"event.score > 3.14", 3.15, true},  // decimal ordering
	}
	for _, c := range cases {
		env := Env{Event: map[string]any{"count": c.val, "score": c.val}}
		if got := evalCond(t, c.src, env); got != c.want {
			t.Errorf("Eval(%q) with val %v(%T) = %v, want %v", c.src, c.val, c.val, got, c.want)
		}
	}
}

// TestConditionStringMatrix covers EDR-010 string comparison: ==/!= lexical;
// ordering ops ⇒ false.
func TestConditionStringMatrix(t *testing.T) {
	env := Env{Event: map[string]any{"branch": "main"}}
	cases := []struct {
		src  string
		want bool
	}{
		{"event.branch == 'main'", true},
		{"event.branch == 'dev'", false},
		{"event.branch != 'dev'", true},
		{"event.branch > 'a'", false}, // ordering on strings ⇒ false
		{"event.branch < 'z'", false}, // ordering on strings ⇒ false
		{"event.branch >= 'main'", false},
		{"event.branch <= 'main'", false},
	}
	for _, c := range cases {
		if got := evalCond(t, c.src, env); got != c.want {
			t.Errorf("Eval(%q) = %v, want %v", c.src, got, c.want)
		}
	}
}

// TestConditionBoolMatrix covers EDR-010 bool comparison.
func TestConditionBoolMatrix(t *testing.T) {
	env := Env{Event: map[string]any{"enabled": true}}
	cases := []struct {
		src  string
		want bool
	}{
		{"event.enabled == true", true},
		{"event.enabled == false", false},
		{"event.enabled != false", true},
		{"event.enabled > false", false}, // ordering on bools ⇒ false
		{"event.enabled < true", false},
	}
	for _, c := range cases {
		if got := evalCond(t, c.src, env); got != c.want {
			t.Errorf("Eval(%q) = %v, want %v", c.src, got, c.want)
		}
	}
}

// TestConditionTypeMismatch covers EDR-010 type-mismatch cells: == ⇒ false,
// != ⇒ true, ordering ⇒ false.
func TestConditionTypeMismatch(t *testing.T) {
	// string value vs number literal
	env := Env{Event: map[string]any{"v": "hello"}}
	if evalCond(t, "event.v == 5", env) {
		t.Error("string == number should be false")
	}
	if !evalCond(t, "event.v != 5", env) {
		t.Error("string != number should be true")
	}
	if evalCond(t, "event.v > 5", env) {
		t.Error("string > number should be false")
	}
	// number value vs string literal
	env2 := Env{Event: map[string]any{"v": float64(5)}}
	if evalCond(t, "event.v == 'x'", env2) {
		t.Error("number == string should be false")
	}
	if !evalCond(t, "event.v != 'x'", env2) {
		t.Error("number != string should be true")
	}
	// bool value vs number literal
	env3 := Env{Event: map[string]any{"v": true}}
	if evalCond(t, "event.v == 1", env3) {
		t.Error("bool == number should be false (no coercion)")
	}
}

// TestConditionPrecedence covers && binding tighter than || : a && b || c parses
// as (a && b) || c.
func TestConditionPrecedence(t *testing.T) {
	// (a && b) || c : choose values so the two groupings differ.
	// a=false, b=true, c=true → (F&&T)||T = T ; F&&(T||T)=F. Result T proves (a&&b)||c.
	env := Env{Event: map[string]any{"a": false, "b": true, "c": true}}
	src := "event.a == true && event.b == true || event.c == true"
	if !evalCond(t, src, env) {
		t.Error("a && b || c should parse as (a&&b)||c and evaluate true")
	}
}

// TestConditionNotBinding covers ! binding and parentheses.
func TestConditionNotBinding(t *testing.T) {
	env := Env{Event: map[string]any{"branch": "main"}}
	if evalCond(t, "!(event.branch == 'main')", env) {
		t.Error("!(main==main) should be false")
	}
	if !evalCond(t, "!(event.branch == 'dev')", env) {
		t.Error("!(main==dev) should be true")
	}
	if !evalCond(t, "(event.branch == 'main')", env) {
		t.Error("(main==main) should be true")
	}
}

// TestConditionNestedGrouping covers a C19-style nested expression against a
// populated Env.
func TestConditionNestedGrouping(t *testing.T) {
	src := "(steps.a.status == 'completed' || event.branch == 'main') && steps.b.status == 'completed'"
	env := Env{
		StepStatus: map[string]string{"a": "failed", "b": "completed"},
		Event:      map[string]any{"branch": "main"},
	}
	// (a.failed==completed=F || branch==main=T) && b.completed==T ⇒ T
	if !evalCond(t, src, env) {
		t.Error("C19-style expression should evaluate true")
	}
	// Flip b to running ⇒ whole thing false.
	env.StepStatus["b"] = "running"
	if evalCond(t, src, env) {
		t.Error("with b not completed the expression should be false")
	}
}

// TestConditionMultiSegmentOutputs covers deep output path comparison (C14-style).
func TestConditionMultiSegmentOutputs(t *testing.T) {
	env := Env{StepOutputs: map[string]map[string]any{
		"draft-entry": {"content": "x"},
	}}
	if !evalCond(t, "steps.draft-entry.outputs.content == 'x'", env) {
		t.Error("multi-segment outputs comparison should be true")
	}
}

// TestConditionRejectPositions spot-checks that specific invalid rows report a
// sensible in-bounds Position beyond the corpus conformance loop.
func TestConditionRejectPositions(t *testing.T) {
	cases := []string{
		"event.count + 1 == 5",              // arithmetic
		"len(event.items) > 0",              // function call
		"event.branch == 'ma' + 'in'",       // concatenation
		"steps['x'].status == 'y'",          // bracket notation
		"event.branch == 'main' ? true : x", // ternary
		"event.branch == 'main",             // unterminated string
		"steps.x.status > null",             // null with ordering op
		"steps.x.status",                    // bare path-ref
		"",                                  // empty input
	}
	for _, src := range cases {
		_, err := ParseCondition(src)
		pe, ok := err.(*ParseError)
		if !ok {
			t.Errorf("ParseCondition(%q): want *ParseError, got %T (%v)", src, err, err)
			continue
		}
		if pe.Position < 0 || pe.Position > len(src) {
			t.Errorf("ParseCondition(%q): Position %d out of bounds [0,%d]", src, pe.Position, len(src))
		}
	}
}
