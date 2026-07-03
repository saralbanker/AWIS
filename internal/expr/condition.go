package expr

import (
	"fmt"
	"strconv"
)

// ConditionExpr is a parsed boolean condition. Grammar (TDS-03 §2, frozen;
// precedence || < && < ! ; parentheses group):
//
//	condition    ::= or-expr
//	or-expr      ::= and-expr | or-expr "||" and-expr
//	and-expr     ::= not-expr | and-expr "&&" not-expr
//	not-expr     ::= compare-expr | "!" not-expr | "(" condition ")"
//	compare-expr ::= path-ref compare-op value | path-ref "==" "null" | path-ref "!=" "null"
//	compare-op   ::= "==" | "!=" | ">" | "<" | ">=" | "<="
//	path-ref     ::= scope "." identifier ("." identifier)*
//	value        ::= string-lit | number-lit | bool-lit
//	scope        ::= "workflow" | "steps" | "event"
type ConditionExpr struct {
	root condNode
}

// condNode is one node in the condition AST.
type condNode interface {
	eval(env Env) (bool, error)
}

type orNode struct{ l, r condNode }
type andNode struct{ l, r condNode }
type notNode struct{ x condNode }

// Value kinds for a compareNode's right-hand literal.
const (
	kindNull = iota
	kindString
	kindNumber
	kindBool
)

type compareNode struct {
	segs    []string // left path-ref segments (segs[0] is the scope)
	op      string   // one of == != > < >= <=
	valKind int      // kindNull | kindString | kindNumber | kindBool
	valStr  string
	valNum  float64
	valBool bool
}

// ParseCondition parses src under the frozen condition grammar, returning a
// *ParseError (as error) with a precise in-bounds Position for every rejection:
// arithmetic (C20), function calls (C21), string concatenation (C22), bracket
// notation (C23), ternary (C24), unterminated string (C25), null with an
// ordering op (C26), a bare path-ref (C27), trailing garbage, and empty input.
// The event scope IS grammatically valid here (corpus C1–C9); the "event only in
// trigger filters" restriction is enforced by the validator (M05-C2), not here.
func ParseCondition(src string) (*ConditionExpr, error) {
	toks, perr := lexCondition(src)
	if perr != nil {
		return nil, perr
	}
	p := &condParser{toks: toks}
	node, perr := p.parseOr()
	if perr != nil {
		return nil, perr
	}
	if p.peek().kind != tkEOF {
		return nil, &ParseError{Position: p.peek().pos, Msg: "unexpected trailing input after condition"}
	}
	return &ConditionExpr{root: node}, nil
}

// Eval evaluates the condition against env. It returns an error only on
// impossible internal states (never for a type mismatch — see the frozen null
// rules and EDR-010 strict semantics implemented in compareNode.eval); in
// practice the error is always nil, but the signature is kept for the M06
// contract.
func (ce *ConditionExpr) Eval(env Env) (bool, error) {
	return ce.root.eval(env)
}

// ---- Parser ----------------------------------------------------------------

type condParser struct {
	toks []token
	i    int
}

func (p *condParser) peek() token { return p.toks[p.i] }
func (p *condParser) next() token {
	t := p.toks[p.i]
	if p.i < len(p.toks)-1 {
		p.i++
	}
	return t
}

func (p *condParser) parseOr() (condNode, *ParseError) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tkOr {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &orNode{l: left, r: right}
	}
	return left, nil
}

func (p *condParser) parseAnd() (condNode, *ParseError) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tkAnd {
		p.next()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &andNode{l: left, r: right}
	}
	return left, nil
}

func (p *condParser) parseNot() (condNode, *ParseError) {
	switch p.peek().kind {
	case tkNot:
		p.next()
		inner, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &notNode{x: inner}, nil
	case tkLParen:
		p.next()
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tkRParen {
			return nil, &ParseError{Position: p.peek().pos, Msg: "expected ')' to close group"}
		}
		p.next()
		return inner, nil
	default:
		return p.parseCompare()
	}
}

func (p *condParser) parseCompare() (condNode, *ParseError) {
	lt := p.peek()
	if lt.kind != tkPath {
		return nil, &ParseError{Position: lt.pos, Msg: "expected a path-ref (scope.identifier...) on the left of a comparison"}
	}
	p.next()

	opTok := p.peek()
	op, ok := opString(opTok.kind)
	if !ok {
		return nil, &ParseError{Position: opTok.pos, Msg: "expected a comparison operator (== != > < >= <=); a bare path-ref is not a condition"}
	}
	p.next()

	node := &compareNode{segs: lt.segs, op: op}
	vt := p.peek()
	switch vt.kind {
	case tkNull:
		if op != "==" && op != "!=" {
			return nil, &ParseError{Position: vt.pos, Msg: "null may only be compared with == or !="}
		}
		node.valKind = kindNull
		p.next()
	case tkString:
		node.valKind = kindString
		node.valStr = vt.str
		p.next()
	case tkNumber:
		node.valKind = kindNumber
		node.valNum = vt.num
		p.next()
	case tkBool:
		node.valKind = kindBool
		node.valBool = vt.b
		p.next()
	default:
		return nil, &ParseError{Position: vt.pos, Msg: "expected a value (string, number, bool, or null) on the right of a comparison"}
	}
	return node, nil
}

// opString maps a comparison token kind to its operator string.
func opString(k tokKind) (string, bool) {
	switch k {
	case tkEQ:
		return "==", true
	case tkNEQ:
		return "!=", true
	case tkGT:
		return ">", true
	case tkLT:
		return "<", true
	case tkGE:
		return ">=", true
	case tkLE:
		return "<=", true
	default:
		return "", false
	}
}

// ---- Evaluation ------------------------------------------------------------

func (n *orNode) eval(env Env) (bool, error) {
	l, err := n.l.eval(env)
	if err != nil {
		return false, err
	}
	r, err := n.r.eval(env)
	if err != nil {
		return false, err
	}
	return l || r, nil
}

func (n *andNode) eval(env Env) (bool, error) {
	l, err := n.l.eval(env)
	if err != nil {
		return false, err
	}
	r, err := n.r.eval(env)
	if err != nil {
		return false, err
	}
	return l && r, nil
}

func (n *notNode) eval(env Env) (bool, error) {
	v, err := n.x.eval(env)
	if err != nil {
		return false, err
	}
	return !v, nil
}

// eval implements the frozen null rules and EDR-010 strict comparison.
func (n *compareNode) eval(env Env) (bool, error) {
	v, ok := env.lookup(n.segs)
	isNull := !ok || v == nil

	// null literal: only == / != reach here (parser rejects ordering vs null).
	if n.valKind == kindNull {
		if n.op == "==" {
			return isNull, nil // null == null ⇒ true
		}
		return !isNull, nil // path != null ⇒ true iff present & non-null
	}

	// Path resolves to null/missing vs a non-null literal (frozen null rule).
	if isNull {
		switch n.op {
		case "==":
			return false, nil // null != non-null
		case "!=":
			return true, nil
		default:
			return false, nil // ordering vs null ⇒ false
		}
	}

	// Both sides present: strict, type-directed comparison (EDR-010, no coercion).
	switch n.valKind {
	case kindNumber:
		f, ok := toFloat(v)
		if !ok {
			return mismatch(n.op), nil
		}
		return numCmp(f, n.valNum, n.op), nil
	case kindString:
		s, ok := v.(string)
		if !ok {
			return mismatch(n.op), nil
		}
		return strCmp(s, n.valStr, n.op), nil
	case kindBool:
		b, ok := v.(bool)
		if !ok {
			return mismatch(n.op), nil
		}
		return boolCmp(b, n.valBool, n.op), nil
	default:
		return false, nil
	}
}

// mismatch is the EDR-010 outcome for type-mismatched operands: == ⇒ false,
// != ⇒ true, ordering ⇒ false.
func mismatch(op string) bool {
	switch op {
	case "==":
		return false
	case "!=":
		return true
	default:
		return false
	}
}

// numCmp compares two float64s under op (all six operators defined for numbers).
func numCmp(a, b float64, op string) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	default:
		return false
	}
}

// strCmp compares strings: only == / != are meaningful; ordering ops ⇒ false.
func strCmp(a, b, op string) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	default:
		return false
	}
}

// boolCmp compares bools: only == / != are meaningful; ordering ops ⇒ false.
func boolCmp(a, b bool, op string) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	default:
		return false
	}
}

// toFloat converts a numeric env value (JSON numbers arrive as float64; Go
// int/int64/float32 etc. are also accepted) to float64. Non-numeric ⇒ ok=false.
func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	default:
		return 0, false
	}
}

// ---- Lexer -----------------------------------------------------------------

type tokKind int

const (
	tkEOF tokKind = iota
	tkPath
	tkString
	tkNumber
	tkBool
	tkNull
	tkEQ
	tkNEQ
	tkGT
	tkLT
	tkGE
	tkLE
	tkAnd
	tkOr
	tkNot
	tkLParen
	tkRParen
)

type token struct {
	kind tokKind
	pos  int
	segs []string // tkPath
	str  string   // tkString
	num  float64  // tkNumber
	b    bool     // tkBool
}

// lexCondition tokenizes src. Whitespace between tokens is skipped. Identifiers
// (including hyphens) are single tokens; true/false/null are keywords; any other
// identifier-starting run is parsed as a path-ref (scope validated). A trailing
// tkEOF token is always appended.
func lexCondition(src string) ([]token, *ParseError) {
	var toks []token
	i, n := 0, len(src)
	for i < n {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '(':
			toks = append(toks, token{kind: tkLParen, pos: i})
			i++
		case c == ')':
			toks = append(toks, token{kind: tkRParen, pos: i})
			i++
		case c == '!':
			if i+1 < n && src[i+1] == '=' {
				toks = append(toks, token{kind: tkNEQ, pos: i})
				i += 2
			} else {
				toks = append(toks, token{kind: tkNot, pos: i})
				i++
			}
		case c == '=':
			if i+1 < n && src[i+1] == '=' {
				toks = append(toks, token{kind: tkEQ, pos: i})
				i += 2
			} else {
				return nil, &ParseError{Position: i, Msg: "unexpected '='; did you mean '=='?"}
			}
		case c == '>':
			if i+1 < n && src[i+1] == '=' {
				toks = append(toks, token{kind: tkGE, pos: i})
				i += 2
			} else {
				toks = append(toks, token{kind: tkGT, pos: i})
				i++
			}
		case c == '<':
			if i+1 < n && src[i+1] == '=' {
				toks = append(toks, token{kind: tkLE, pos: i})
				i += 2
			} else {
				toks = append(toks, token{kind: tkLT, pos: i})
				i++
			}
		case c == '&':
			if i+1 < n && src[i+1] == '&' {
				toks = append(toks, token{kind: tkAnd, pos: i})
				i += 2
			} else {
				return nil, &ParseError{Position: i, Msg: "unexpected '&'; did you mean '&&'?"}
			}
		case c == '|':
			if i+1 < n && src[i+1] == '|' {
				toks = append(toks, token{kind: tkOr, pos: i})
				i += 2
			} else {
				return nil, &ParseError{Position: i, Msg: "unexpected '|'; did you mean '||'?"}
			}
		case c == '\'':
			j := i + 1
			for j < n && src[j] != '\'' {
				j++
			}
			if j >= n {
				return nil, &ParseError{Position: i, Msg: "unterminated string literal"}
			}
			toks = append(toks, token{kind: tkString, pos: i, str: src[i+1 : j]})
			i = j + 1
		case c >= '0' && c <= '9':
			j := i + 1
			for j < n && src[j] >= '0' && src[j] <= '9' {
				j++
			}
			if j < n && src[j] == '.' {
				k := j + 1
				if k >= n || src[k] < '0' || src[k] > '9' {
					return nil, &ParseError{Position: j, Msg: "expected a digit after '.' in number literal"}
				}
				for k < n && src[k] >= '0' && src[k] <= '9' {
					k++
				}
				j = k
			}
			f, err := strconv.ParseFloat(src[i:j], 64)
			if err != nil {
				return nil, &ParseError{Position: i, Msg: "invalid number literal"}
			}
			toks = append(toks, token{kind: tkNumber, pos: i, num: f})
			i = j
		case isIdentStart(c):
			tok, np, _ := readIdent(src, i)
			switch tok {
			case "true":
				toks = append(toks, token{kind: tkBool, pos: i, b: true})
				i = np
			case "false":
				toks = append(toks, token{kind: tkBool, pos: i, b: false})
				i = np
			case "null":
				toks = append(toks, token{kind: tkNull, pos: i})
				i = np
			default:
				segs, np2, perr := parsePathRef(src, i, condScopes)
				if perr != nil {
					return nil, perr
				}
				toks = append(toks, token{kind: tkPath, pos: i, segs: segs})
				i = np2
			}
		default:
			return nil, &ParseError{Position: i, Msg: fmt.Sprintf("unexpected character %q", string(c))}
		}
	}
	toks = append(toks, token{kind: tkEOF, pos: n})
	return toks, nil
}
