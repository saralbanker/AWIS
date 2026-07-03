package expr

import (
	"fmt"
	"strconv"
	"strings"
)

// Template is a parsed template string: a sequence of literal-text and path-ref
// parts. Grammar (TDS-03 §1, frozen):
//
//	template-string ::= literal-text | "{{" path-ref "}}" | template-string template-string
//	path-ref        ::= scope "." identifier ("." identifier)*
//	scope           ::= "workflow" | "steps"
//	identifier      ::= [a-zA-Z_][a-zA-Z0-9_-]*
type Template struct {
	parts []tmplPart
}

// tmplPart is one segment of a parsed template: either a literal (isRef false) or
// a path-ref (isRef true).
type tmplPart struct {
	lit   string   // literal text when isRef is false
	isRef bool     // true for a {{ path-ref }} part
	segs  []string // path segments when isRef is true (segs[0] is the scope)
	path  string   // dotted path form (e.g. "workflow.inputs.name") for Warning.Path
}

// Warning records a template reference that resolved to the empty string at
// execution time (a missing path or a not-yet-completed step). Warnings are
// returned as data — this package owns no logger; the engine logs them at M06
// (FR-WD-05).
type Warning struct {
	Path   string
	Reason string
}

// ParseTemplate parses src under the frozen template grammar. It returns a
// *ParseError (as error) for every prohibited construct — arithmetic inside
// {{}}, function calls, {{if …}}, nested templates, bracket notation, a scope
// other than workflow|steps (e.g. {{event.branch}}), and a bare scope
// {{workflow}} — each with a precise in-bounds Position (corpus rows T9–T15).
//
// Whitespace decision (documented per card, contradicts no corpus row): leading
// and trailing ASCII spaces INSIDE the braces are tolerated (a lexer skip), since
// YAML authors will write "{{ x }}". The grammar production "{{" path-ref "}}"
// has no whitespace terminals, so anything other than surrounding spaces is
// non-conforming and rejected.
//
// Literal handling: everything outside "{{" is literal-text. A "}}" with no
// matching opener is literal (the grammar treats it as literal-text; no corpus
// row contradicts). An unmatched "{{" (no closing "}}") is a ParseError whose
// Position is the offset of the "{{".
func ParseTemplate(src string) (*Template, error) {
	var parts []tmplPart
	i, n := 0, len(src)
	for i < n {
		rel := strings.Index(src[i:], "{{")
		if rel < 0 {
			parts = append(parts, tmplPart{lit: src[i:]})
			break
		}
		open := i + rel
		if open > i {
			parts = append(parts, tmplPart{lit: src[i:open]})
		}
		segs, next, perr := parseTemplateRef(src, open)
		if perr != nil {
			return nil, perr
		}
		parts = append(parts, tmplPart{isRef: true, segs: segs, path: strings.Join(segs, ".")})
		i = next
	}
	return &Template{parts: parts}, nil
}

// parseTemplateRef parses a single "{{" path-ref "}}" beginning at open (the
// offset of the first '{'). It returns the path segments and the position just
// past the closing "}}".
func parseTemplateRef(src string, open int) ([]string, int, *ParseError) {
	pos := open + 2 // skip "{{"
	for pos < len(src) && src[pos] == ' ' {
		pos++
	}
	segs, np, perr := parsePathRef(src, pos, templateScopes)
	if perr != nil {
		return nil, 0, perr
	}
	pos = np
	for pos < len(src) && src[pos] == ' ' {
		pos++
	}
	if pos+1 < len(src) && src[pos] == '}' && src[pos+1] == '}' {
		return segs, pos + 2, nil
	}
	if pos >= len(src) {
		return nil, 0, &ParseError{Position: open, Msg: "unclosed '{{' (missing '}}')"}
	}
	return nil, 0, &ParseError{Position: pos, Msg: "expected '}}' to close the template expression"}
}

// Resolve produces the concrete string for this template against env, plus any
// warnings. FROZEN semantics (FR-WD-05): a referenced path that is missing, OR a
// referenced step that has not yet completed, contributes "" and appends one
// Warning — resolution NEVER fails (no error return). "Not yet completed"
// concretely: for steps.<id>.outputs.…, if StepStatus[id] exists and is not
// "completed", the ref yields ""+Warning even when outputs are present. A
// steps.<id>.status read is not gated on completion (it returns whatever status
// exists). Non-string resolved values stringify deterministically (see
// stringify, EDR-010).
func (t *Template) Resolve(env Env) (string, []Warning) {
	var sb strings.Builder
	var warns []Warning
	for _, p := range t.parts {
		if !p.isRef {
			sb.WriteString(p.lit)
			continue
		}
		v, ok, reason := resolveRef(env, p.segs)
		if !ok {
			warns = append(warns, Warning{Path: p.path, Reason: reason})
			continue // contributes ""
		}
		sb.WriteString(stringify(v))
	}
	return sb.String(), warns
}

// resolveRef reads one path from env, applying the FR-WD-05 status gate for
// output references. It returns (value, ok, reason); when ok is false the ref
// resolves to "" and reason explains the warning.
func resolveRef(env Env, segs []string) (any, bool, string) {
	// Gate outputs on step completion: steps.<id>.outputs.…
	if len(segs) >= 4 && segs[0] == "steps" && segs[2] == "outputs" {
		id := segs[1]
		if st, ok := env.StepStatus[id]; ok && st != "completed" {
			return nil, false, fmt.Sprintf("step %q has not completed (status %q)", id, st)
		}
	}
	v, ok := env.lookup(segs)
	if !ok {
		return nil, false, "referenced path is missing or resolves to null"
	}
	return v, true, ""
}

// stringify renders a resolved value deterministically (EDR-010): bool →
// "true"/"false"; JSON/Go numbers → strconv (floats use shortest round-trip via
// FormatFloat with precision -1); string → itself; nil → ""; anything else
// (composite / unknown) → fmt.Sprintf("%v").
func stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(x), 'g', -1, 32)
	case int:
		return strconv.Itoa(x)
	case int8:
		return strconv.FormatInt(int64(x), 10)
	case int16:
		return strconv.FormatInt(int64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint:
		return strconv.FormatUint(uint64(x), 10)
	case uint8:
		return strconv.FormatUint(uint64(x), 10)
	case uint16:
		return strconv.FormatUint(uint64(x), 10)
	case uint32:
		return strconv.FormatUint(uint64(x), 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}
