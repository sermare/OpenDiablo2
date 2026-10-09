package d2calc

import (
	"strings"
)

// Kind selects the field-code table a string is compiled against: skillcalc.txt
// (skills.txt, skilldesc.txt) or misscalc.txt (missiles.txt).
type Kind int

// The two calc dialects.
const (
	KindSkill Kind = iota
	KindMissile
)

// Env supplies everything an expression can read. Field receives the
// normalised (lower case, 4 character) code of a bare identifier such as
// "lvl", "par3", "ln12" or "edmn".
type Env interface {
	// Field evaluates a bare field code for the context skill/missile+level.
	Field(code string) int
	// Skill evaluates skill('name'.field): the field of another skill at
	// that skill's own level for the context unit (0 if the unit lacks it).
	Skill(name, field string) int
	// Miss evaluates miss('name'.field) for a missile record.
	Miss(name, field string) int
	// Stat evaluates stat('name'.mode); mode is base, mod or accr.
	Stat(name, mode string) int
	// Sklvl evaluates sklvl(a, b, c). Argument meaning is UNVERIFIED (the
	// notes: field b of skill a at the level given by field c of the current
	// skill); it is unused by the shipped tables.
	Sklvl(a, b, c int) int
	// Rand rolls a value for rand(a, b); it is only called when b > a.
	Rand(a, b int) int
}

// ZeroEnv is an Env that returns 0 for everything; embed it to implement a
// subset.
type ZeroEnv struct{}

// Field implements Env.
func (ZeroEnv) Field(string) int { return 0 }

// Skill implements Env.
func (ZeroEnv) Skill(string, string) int { return 0 }

// Miss implements Env.
func (ZeroEnv) Miss(string, string) int { return 0 }

// Stat implements Env.
func (ZeroEnv) Stat(string, string) int { return 0 }

// Sklvl implements Env.
func (ZeroEnv) Sklvl(int, int, int) int { return 0 }

// Rand implements Env.
func (ZeroEnv) Rand(a, _ int) int { return a }

type opCode uint8

const (
	opNum opCode = iota
	opField
	opRefSkill
	opRefMiss
	opRefStat
	opFunc
	opLT
	opLE
	opGT
	opGE
	opEQ
	opNE
	opAdd
	opSub
	opMul
	opDiv
	opPow
	opNeg
	opTern
)

type op struct {
	code opCode
	num  int32
	name string // field code, function name, or referenced name
	arg  string // reference field / mode
	argc int
}

// Program is a compiled expression. The zero value and nil evaluate to 0.
type Program struct {
	src string
	ops []op
}

// Source returns the text the program was compiled from.
func (p *Program) Source() string {
	if p == nil {
		return ""
	}

	return p.src
}

// Empty reports whether the program has no code (an empty txt cell).
func (p *Program) Empty() bool { return p == nil || len(p.ops) == 0 }

func (p *Program) String() string { return p.Source() }

// function table: name -> arity
var funcArity = map[string]int{
	"min": 2, "max": 2, "rand": 2, "skill": 2, "miss": 2, "stat": 2, "sklvl": 3,
}

type tokKind uint8

const (
	tNum tokKind = iota
	tStr
	tIdent
	tSym
)

type token struct {
	kind tokKind
	text string
	num  int32
}

func tokenize(s string) []token {
	var toks []token

	i := 0
	for i < len(s) {
		c := s[i]

		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
		case c >= '0' && c <= '9':
			var n int32

			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				n = n*10 + int32(s[i]-'0')
				i++
			}

			toks = append(toks, token{kind: tNum, num: n})
		case c == '\'':
			j := i + 1
			for j < len(s) && s[j] != '\'' {
				j++
			}

			toks = append(toks, token{kind: tStr, text: s[i+1 : min(j, len(s))]})
			i = j + 1
		case isAlpha(c):
			j := i
			for j < len(s) && (isAlpha(s[j]) || (s[j] >= '0' && s[j] <= '9')) {
				j++
			}

			toks = append(toks, token{kind: tIdent, text: s[i:j]})
			i = j
		default:
			sym := string(c)
			if i+1 < len(s) && s[i+1] == '=' && (c == '<' || c == '>' || c == '=' || c == '!') {
				sym = s[i : i+2]
			}

			i += len(sym)

			switch sym {
			case "=", "!": // a lone '=' or '!' is silently ignored by the game
			case "+", "-", "*", "/", "^", "<", "<=", ">", ">=", "==", "!=", "?", ":", "(", ")", ",", ".":
				toks = append(toks, token{kind: tSym, text: sym})
			}
		}
	}

	return toks
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func isAlpha(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// stack item kinds of the shunting-yard operator stack
type stackItem struct {
	paren bool // '(' marker
	fn    string
	op    opCode
	prec  int
	isFn  bool
}

// Operator ranks, loosest to tightest (verified order, see package comment).
func precedence(c opCode) int {
	switch c {
	case opLT, opLE, opGT, opGE, opEQ, opNE:
		return 1
	case opAdd, opSub:
		return 2
	case opMul, opDiv:
		return 3
	case opPow:
		return 4
	case opNeg:
		return 5
	case opTern:
		return 6
	}

	return 0
}

var symOps = map[string]opCode{
	"<": opLT, "<=": opLE, ">": opGT, ">=": opGE, "==": opEQ, "!=": opNE,
	"+": opAdd, "-": opSub, "*": opMul, "/": opDiv, "^": opPow, "?": opTern,
}

// Compile compiles a calc string. It never fails: like the game it ignores
// stray characters, tolerates an unclosed final parenthesis (the original
// Fire Wall calc has one) and turns unknown identifiers into 0.
func Compile(src string, kind Kind) *Program {
	prog := &Program{src: src}
	toks := tokenize(strings.TrimSpace(src))

	known := skillCodes
	if kind == KindMissile {
		known = missileCodes
	}

	var (
		out         []op
		stack       []stackItem
		prevOperand bool
	)

	popTo := func(pred func(it stackItem) bool) {
		for len(stack) > 0 {
			top := stack[len(stack)-1]
			if !pred(top) {
				return
			}

			stack = stack[:len(stack)-1]
			out = append(out, op{code: top.op})
		}
	}

	for i := 0; i < len(toks); i++ {
		t := toks[i]

		switch t.kind {
		case tNum:
			out = append(out, op{code: opNum, num: t.num})
			prevOperand = true
		case tStr:
			// a quoted name outside skill()/miss()/stat() resolves by position in
			// the game (to a skill/missile/stat id); not modelled, UNVERIFIED -> 0.
			out = append(out, op{code: opNum})

			if i+2 < len(toks) && toks[i+1].text == "." && toks[i+2].kind == tIdent {
				i += 2
			}

			prevOperand = true
		case tIdent:
			name := strings.ToLower(t.text)
			isCall := i+1 < len(toks) && toks[i+1].kind == tSym && toks[i+1].text == "("

			if _, isFn := funcArity[name]; isFn && isCall {
				if name == "skill" || name == "miss" || name == "stat" {
					o, next := parseRef(toks, i+2, name)
					out = append(out, o)
					i = next
					prevOperand = true

					continue
				}

				stack = append(stack, stackItem{isFn: true, fn: name})
				i++ // consume '('
				prevOperand = false

				continue
			}

			code := name
			if len(code) > 4 {
				code = code[:4]
			}

			if known[code] {
				out = append(out, op{code: opField, name: code})
			} else {
				out = append(out, op{code: opNum})
			}

			prevOperand = true
		case tSym:
			switch t.text {
			case "(":
				stack = append(stack, stackItem{paren: true})
				prevOperand = false
			case ")":
				popTo(func(it stackItem) bool { return !it.paren && !it.isFn })

				if len(stack) > 0 {
					top := stack[len(stack)-1]
					stack = stack[:len(stack)-1]

					if top.isFn {
						out = append(out, op{code: opFunc, name: top.fn, argc: funcArity[top.fn]})
					}
				}

				prevOperand = true
			case ",":
				popTo(func(it stackItem) bool { return !it.paren && !it.isFn })

				prevOperand = false
			case ":":
				prevOperand = false
			case "-":
				if !prevOperand {
					stack = append(stack, stackItem{op: opNeg, prec: precedence(opNeg)})
					continue
				}

				fallthrough
			case "+":
				if !prevOperand {
					continue // unary plus is ignored
				}

				fallthrough
			default:
				c, ok := symOps[t.text]
				if !ok {
					continue
				}

				p := precedence(c)
				popTo(func(it stackItem) bool { return !it.paren && !it.isFn && it.prec >= p })
				stack = append(stack, stackItem{op: c, prec: p})
				prevOperand = false
			}
		}
	}

	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		switch {
		case top.paren:
		case top.isFn:
			out = append(out, op{code: opFunc, name: top.fn, argc: funcArity[top.fn]})
		default:
			out = append(out, op{code: top.op})
		}
	}

	prog.ops = out

	return prog
}

// parseRef parses 'Name'.field ) starting at toks[i]; it returns the op and
// the index of the closing parenthesis (or the last token consumed).
func parseRef(toks []token, i int, fn string) (op, int) {
	code := map[string]opCode{"skill": opRefSkill, "miss": opRefMiss, "stat": opRefStat}[fn]

	if i+3 < len(toks) && toks[i].kind == tStr && toks[i+1].text == "." && toks[i+2].kind == tIdent &&
		toks[i+3].text == ")" {
		field := strings.ToLower(toks[i+2].text)
		if fn != "stat" && len(field) > 4 {
			field = field[:4]
		}

		return op{code: code, name: toks[i].text, arg: field}, i + 3
	}

	// malformed reference: skip to the closing parenthesis, yield 0
	depth := 1
	j := i

	for ; j < len(toks) && depth > 0; j++ {
		switch toks[j].text {
		case "(":
			depth++
		case ")":
			depth--
		}
	}

	return op{code: opNum}, j - 1
}

// Eval evaluates the program against env and returns the top of the stack
// (0 for an empty program or an empty stack). Arithmetic is 32 bit signed.
func (p *Program) Eval(env Env) int {
	if p == nil {
		return 0
	}

	const maxDepth = 64

	var st [maxDepth]int32

	sp := 0

	push := func(v int32) {
		if sp < maxDepth {
			st[sp] = v
			sp++
		}
	}
	pop := func() int32 {
		if sp == 0 {
			return 0
		}

		sp--

		return st[sp]
	}

	for _, o := range p.ops {
		switch o.code {
		case opNum:
			push(o.num)
		case opField:
			push(int32(env.Field(o.name)))
		case opRefSkill:
			push(int32(env.Skill(o.name, o.arg)))
		case opRefMiss:
			push(int32(env.Miss(o.name, o.arg)))
		case opRefStat:
			push(int32(env.Stat(o.name, o.arg)))
		case opFunc:
			push(callFunc(env, o, pop))
		case opNeg:
			push(-pop())
		case opTern:
			c, b, a := pop(), pop(), pop()
			if a != 0 {
				push(b)
			} else {
				push(c)
			}
		default:
			r, l := pop(), pop()
			push(binary(o.code, l, r))
		}
	}

	return int(pop())
}

func callFunc(env Env, o op, pop func() int32) int32 {
	switch o.name {
	case "min":
		b, a := pop(), pop()
		if a < b {
			return a
		}

		return b
	case "max":
		b, a := pop(), pop()
		if a > b {
			return a
		}

		return b
	case "rand":
		b, a := pop(), pop()
		if b <= a {
			return a
		}

		return int32(env.Rand(int(a), int(b)))
	case "sklvl":
		c, b, a := pop(), pop(), pop()

		return int32(env.Sklvl(int(a), int(b), int(c)))
	}

	return 0
}

func binary(c opCode, l, r int32) int32 {
	b2i := func(b bool) int32 {
		if b {
			return 1
		}

		return 0
	}

	switch c {
	case opLT:
		return b2i(l < r)
	case opLE:
		return b2i(l <= r)
	case opGT:
		return b2i(l > r)
	case opGE:
		return b2i(l >= r)
	case opEQ:
		return b2i(l == r)
	case opNE:
		return b2i(l != r)
	case opAdd:
		return l + r
	case opSub:
		return l - r
	case opMul:
		return l * r
	case opDiv:
		if r == 0 {
			return 0 // division by zero pushes 0 (verified)
		}

		return l / r
	case opPow:
		if r <= 0 {
			return 1
		}

		res := int32(1)

		for i := int32(0); i < r && i < 1<<16; i++ {
			res *= l
		}

		return res
	}

	return 0
}
