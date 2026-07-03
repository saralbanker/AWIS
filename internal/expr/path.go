package expr

import "fmt"

// Scope sets, per the frozen grammars (TDS-03). Templates allow only
// workflow|steps; conditions additionally allow event (the field-context
// restriction "event only in trigger filters" is the validator's job at M05-C2,
// not the parser's).
var (
	templateScopes = map[string]bool{"workflow": true, "steps": true}
	condScopes     = map[string]bool{"workflow": true, "steps": true, "event": true}
)

// isIdentStart reports whether c may begin an identifier: [a-zA-Z_].
func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isIdentPart reports whether c may continue an identifier: [a-zA-Z0-9_-].
// Hyphens are identifier characters (single tokens), NEVER subtraction — this is
// the frozen rule that makes steps.draft-entry.status canonical.
func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9') || c == '-'
}

// readIdent reads one identifier token ([a-zA-Z_][a-zA-Z0-9_-]*) starting at pos.
// It returns the token, the position just past it, and ok=false if pos is not at
// an identifier-start byte.
func readIdent(src string, pos int) (string, int, bool) {
	if pos >= len(src) || !isIdentStart(src[pos]) {
		return "", pos, false
	}
	j := pos + 1
	for j < len(src) && isIdentPart(src[j]) {
		j++
	}
	return src[pos:j], j, true
}

// parsePathRef parses  scope "." identifier ("." identifier)*  starting at pos,
// validating scope against allowed. It is shared by both grammars (template ref
// interiors and condition path operands). It returns the path segments (segs[0]
// is the scope), the position just past the path, or a ParseError with a precise
// in-bounds Position. A bare scope with no following identifier is rejected
// (e.g. {{workflow}} — corpus T15).
func parsePathRef(src string, pos int, allowed map[string]bool) ([]string, int, *ParseError) {
	scope, np, ok := readIdent(src, pos)
	if !ok {
		return nil, 0, &ParseError{Position: pos, Msg: "expected a scope identifier (workflow, steps, or event)"}
	}
	if !allowed[scope] {
		return nil, 0, &ParseError{Position: pos, Msg: fmt.Sprintf("invalid scope %q for this context", scope)}
	}
	segs := []string{scope}
	pos = np
	// path-ref requires at least one ".identifier" after the scope.
	if pos >= len(src) || src[pos] != '.' {
		return nil, 0, &ParseError{Position: pos, Msg: fmt.Sprintf("scope %q must be followed by '.' and at least one identifier", scope)}
	}
	for pos < len(src) && src[pos] == '.' {
		pos++ // consume '.'
		id, np2, ok := readIdent(src, pos)
		if !ok {
			return nil, 0, &ParseError{Position: pos, Msg: "expected an identifier after '.'"}
		}
		segs = append(segs, id)
		pos = np2
	}
	return segs, pos, nil
}
