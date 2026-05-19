package dsqlex

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/govalues/decimal"
)

// Value types
type ValueType int

const (
	ValDecimal ValueType = iota
	ValString
	ValBool
	ValNull
)

type Value struct {
	Type   ValueType
	DecVal decimal.Decimal
	StrVal string
	BoolV  bool
}

var NullValue = Value{Type: ValNull}
var TrueValue = Value{Type: ValBool, BoolV: true}
var FalseValue = Value{Type: ValBool, BoolV: false}

func DecimalValue(d decimal.Decimal) Value {
	return Value{Type: ValDecimal, DecVal: d}
}

func StringValue(s string) Value {
	return Value{Type: ValString, StrVal: s}
}

func BoolValue(b bool) Value {
	if b {
		return TrueValue
	}
	return FalseValue
}

// Context holds variable bindings.
type Context struct {
	Fields map[string]Value
	Nested map[string]*Context
}

func NewContext() *Context {
	return &Context{
		Fields: make(map[string]Value),
	}
}

func (c *Context) SetDecimal(key, val string) {
	d, _ := decimal.Parse(val)
	c.Fields[key] = DecimalValue(d)
}

func (c *Context) SetString(key, val string) {
	c.Fields[key] = StringValue(val)
}

func (c *Context) SetBool(key string, val bool) {
	c.Fields[key] = BoolValue(val)
}

func (c *Context) SetNull(key string) {
	c.Fields[key] = NullValue
}

func (c *Context) SetNested(key string, ctx *Context) {
	if c.Nested == nil {
		c.Nested = make(map[string]*Context)
	}
	c.Nested[key] = ctx
}

// EvalOptions for resolvers.
type EvalOptions struct {
	Resolver      func(name string, visited map[string]bool) (Value, error)
	EventResolver func(typ, subtype string, ctx *Context, visited map[string]bool) (Value, error)
	Visited       map[string]bool
}

// ── helpers ──

func isTruthy(v Value) bool {
	if v.Type == ValNull {
		return false
	}
	if v.Type == ValBool && !v.BoolV {
		return false
	}
	return true
}

func valueToDecimal(v Value) (decimal.Decimal, error) {
	switch v.Type {
	case ValDecimal:
		return v.DecVal, nil
	case ValString:
		d, err := decimal.Parse(v.StrVal)
		if err != nil {
			return decimal.Decimal{}, fmt.Errorf("cannot convert '%s' to decimal", v.StrVal)
		}
		return d, nil
	case ValBool:
		return decimal.Decimal{}, fmt.Errorf("cannot convert boolean to decimal")
	case ValNull:
		return decimal.Decimal{}, fmt.Errorf("cannot convert NULL to decimal")
	}
	return decimal.Decimal{}, fmt.Errorf("unknown value type")
}

func valToString(v Value) string {
	switch v.Type {
	case ValDecimal:
		return v.DecVal.String()
	case ValString:
		return v.StrVal
	case ValBool:
		if v.BoolV {
			return "TRUE"
		}
		return "FALSE"
	case ValNull:
		return "NULL"
	}
	return ""
}

// compareValues returns -1, 0, 1, or -2 (not comparable)
func compareValues(lhs, rhs Value) int {
	if lhs.Type == ValNull && rhs.Type == ValNull {
		return 0
	}
	if lhs.Type == ValNull || rhs.Type == ValNull {
		return -2
	}
	if lhs.Type == ValDecimal && rhs.Type == ValDecimal {
		return lhs.DecVal.Cmp(rhs.DecVal)
	}
	if lhs.Type == ValString && rhs.Type == ValString {
		// Try decimal conversion
		dl, errL := decimal.Parse(lhs.StrVal)
		dr, errR := decimal.Parse(rhs.StrVal)
		if errL == nil && errR == nil {
			return dl.Cmp(dr)
		}
		return strings.Compare(lhs.StrVal, rhs.StrVal)
	}
	if lhs.Type == ValBool && rhs.Type == ValBool {
		a, b := 0, 0
		if lhs.BoolV {
			a = 1
		}
		if rhs.BoolV {
			b = 1
		}
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
	// Mixed: try decimal
	dl, errL := valueToDecimal(lhs)
	dr, errR := valueToDecimal(rhs)
	if errL == nil && errR == nil {
		return dl.Cmp(dr)
	}
	sl := valToString(lhs)
	sr := valToString(rhs)
	return strings.Compare(sl, sr)
}

func likeMatch(text, pattern string) bool {
	var b strings.Builder
	b.WriteString("(?i)^")
	for _, ch := range pattern {
		switch ch {
		case '%':
			b.WriteString(".*")
		case '_':
			b.WriteByte('.')
		default:
			if strings.ContainsRune(".+*?^${}()|[]\\", ch) {
				b.WriteByte('\\')
			}
			b.WriteRune(ch)
		}
	}
	b.WriteByte('$')
	re, err := regexp.Compile(b.String())
	if err != nil {
		return false
	}
	return re.MatchString(text)
}

// ── resolve ──

func resolveIdentifier(name string, ctx *Context, opts *EvalOptions) (Value, error) {
	if v, ok := ctx.Fields[name]; ok {
		return v, nil
	}

	// Dot-path
	if dot := strings.IndexByte(name, '.'); dot >= 0 {
		first := name[:dot]
		rest := name[dot+1:]
		if ctx.Nested != nil {
			if nested, ok := ctx.Nested[first]; ok {
				return resolveIdentifier(rest, nested, opts)
			}
		}
	}

	if opts != nil && opts.Resolver != nil {
		visited := opts.Visited
		if visited == nil {
			visited = make(map[string]bool)
		}
		return opts.Resolver(name, visited)
	}

	return NullValue, fmt.Errorf("unknown field: %s", name)
}

// ── main evaluate ──

func Evaluate(ast *AstNode, ctx *Context, opts *EvalOptions) (Value, error) {
	switch ast.Kind {
	case NodeSelect:
		return Evaluate(ast.Expr, ctx, opts)

	case NodeNumberLit:
		return DecimalValue(ast.DecVal), nil

	case NodeStringLit:
		return StringValue(ast.StrVal), nil

	case NodeBoolLit:
		return BoolValue(ast.BoolVal), nil

	case NodeNullLit:
		return NullValue, nil

	case NodeIdentifier:
		return resolveIdentifier(ast.StrVal, ctx, opts)

	case NodeBinaryOp:
		return evalBinop(ast, ctx, opts)

	case NodeCaseExpr:
		for _, wc := range ast.Whens {
			cond, err := Evaluate(wc.Condition, ctx, opts)
			if err != nil {
				return NullValue, err
			}
			if isTruthy(cond) {
				return Evaluate(wc.Result, ctx, opts)
			}
		}
		if ast.ElseClause != nil {
			return Evaluate(ast.ElseClause, ctx, opts)
		}
		return NullValue, nil

	case NodeFunctionCall:
		return evalFunction(ast.StrVal, ast.Args, ctx, opts)

	case NodeInExpr:
		val, err := Evaluate(ast.Expr, ctx, opts)
		if err != nil {
			return NullValue, err
		}
		for _, item := range ast.Args {
			iv, err := Evaluate(item, ctx, opts)
			if err != nil {
				return NullValue, err
			}
			if compareValues(val, iv) == 0 {
				return TrueValue, nil
			}
		}
		return FalseValue, nil

	case NodeNotInExpr:
		val, err := Evaluate(ast.Expr, ctx, opts)
		if err != nil {
			return NullValue, err
		}
		for _, item := range ast.Args {
			iv, err := Evaluate(item, ctx, opts)
			if err != nil {
				return NullValue, err
			}
			if compareValues(val, iv) == 0 {
				return FalseValue, nil
			}
		}
		return TrueValue, nil

	case NodeLikeExpr:
		val, err := Evaluate(ast.Expr, ctx, opts)
		if err != nil {
			return NullValue, err
		}
		pat, err := Evaluate(ast.Pattern, ctx, opts)
		if err != nil {
			return NullValue, err
		}
		if val.Type == ValNull || pat.Type == ValNull {
			return NullValue, nil
		}
		return BoolValue(likeMatch(valToString(val), valToString(pat))), nil

	case NodeNotLikeExpr:
		val, err := Evaluate(ast.Expr, ctx, opts)
		if err != nil {
			return NullValue, err
		}
		pat, err := Evaluate(ast.Pattern, ctx, opts)
		if err != nil {
			return NullValue, err
		}
		if val.Type == ValNull || pat.Type == ValNull {
			return NullValue, nil
		}
		return BoolValue(!likeMatch(valToString(val), valToString(pat))), nil
	}

	return NullValue, fmt.Errorf("unknown node kind: %d", ast.Kind)
}

func evalBinop(node *AstNode, ctx *Context, opts *EvalOptions) (Value, error) {
	// Short-circuit AND/OR
	switch node.Op {
	case OpAnd:
		lv, err := Evaluate(node.Left, ctx, opts)
		if err != nil {
			return NullValue, err
		}
		if !isTruthy(lv) {
			return lv, nil
		}
		return Evaluate(node.Right, ctx, opts)
	case OpOr:
		lv, err := Evaluate(node.Left, ctx, opts)
		if err != nil {
			return NullValue, err
		}
		if isTruthy(lv) {
			return lv, nil
		}
		return Evaluate(node.Right, ctx, opts)
	}

	lv, err := Evaluate(node.Left, ctx, opts)
	if err != nil {
		return NullValue, err
	}
	rv, err := Evaluate(node.Right, ctx, opts)
	if err != nil {
		return NullValue, err
	}

	switch node.Op {
	case OpPlus, OpMinus, OpMultiply, OpDivide:
		ld, err := valueToDecimal(lv)
		if err != nil {
			return NullValue, err
		}
		rd, err := valueToDecimal(rv)
		if err != nil {
			return NullValue, err
		}
		var result decimal.Decimal
		var opErr error
		switch node.Op {
		case OpPlus:
			result, opErr = ld.Add(rd)
		case OpMinus:
			result, opErr = ld.Sub(rd)
		case OpMultiply:
			result, opErr = ld.Mul(rd)
		case OpDivide:
			if rd.IsZero() {
				return NullValue, fmt.Errorf("division by zero")
			}
			result, opErr = ld.Quo(rd)
		}
		if opErr != nil {
			return NullValue, opErr
		}
		return DecimalValue(result), nil

	case OpEq:
		// Fast path for same types
		if lv.Type == ValString && rv.Type == ValString {
			return BoolValue(lv.StrVal == rv.StrVal), nil
		}
		if lv.Type == ValDecimal && rv.Type == ValDecimal {
			return BoolValue(lv.DecVal.Cmp(rv.DecVal) == 0), nil
		}
		return BoolValue(compareValues(lv, rv) == 0), nil

	case OpNeq:
		if lv.Type == ValString && rv.Type == ValString {
			return BoolValue(lv.StrVal != rv.StrVal), nil
		}
		if lv.Type == ValDecimal && rv.Type == ValDecimal {
			return BoolValue(lv.DecVal.Cmp(rv.DecVal) != 0), nil
		}
		return BoolValue(compareValues(lv, rv) != 0), nil

	case OpLt:
		return BoolValue(compareValues(lv, rv) == -1), nil
	case OpGt:
		return BoolValue(compareValues(lv, rv) == 1), nil
	case OpLte:
		c := compareValues(lv, rv)
		return BoolValue(c == 0 || c == -1), nil
	case OpGte:
		c := compareValues(lv, rv)
		return BoolValue(c == 0 || c == 1), nil
	}

	return NullValue, fmt.Errorf("unknown operator: %d", node.Op)
}

func evalFunction(name string, args []*AstNode, ctx *Context, opts *EvalOptions) (Value, error) {
	switch name {
	case "ROUND":
		if len(args) != 2 {
			return NullValue, fmt.Errorf("ROUND requires exactly 2 arguments")
		}
		val, err := Evaluate(args[0], ctx, opts)
		if err != nil {
			return NullValue, err
		}
		if val.Type == ValNull {
			return NullValue, nil
		}
		d, err := valueToDecimal(val)
		if err != nil {
			return NullValue, err
		}
		precVal, err := Evaluate(args[1], ctx, opts)
		if err != nil {
			return NullValue, err
		}
		precD, err := valueToDecimal(precVal)
		if err != nil {
			return NullValue, err
		}
		precLo, _, ok := precD.Int64(0)
		if !ok {
			return NullValue, fmt.Errorf("ROUND precision must be an integer")
		}
		rounded := d.Round(int(precLo))
		return DecimalValue(rounded), nil

	case "COALESCE":
		for _, arg := range args {
			val, err := Evaluate(arg, ctx, opts)
			if err != nil {
				return NullValue, err
			}
			if val.Type != ValNull {
				return val, nil
			}
		}
		return NullValue, nil

	case "UPPER":
		if len(args) != 1 {
			return NullValue, fmt.Errorf("UPPER requires exactly 1 argument")
		}
		val, err := Evaluate(args[0], ctx, opts)
		if err != nil {
			return NullValue, err
		}
		if val.Type == ValNull {
			return NullValue, nil
		}
		return StringValue(strings.ToUpper(valToString(val))), nil

	case "LOWER":
		if len(args) != 1 {
			return NullValue, fmt.Errorf("LOWER requires exactly 1 argument")
		}
		val, err := Evaluate(args[0], ctx, opts)
		if err != nil {
			return NullValue, err
		}
		if val.Type == ValNull {
			return NullValue, nil
		}
		return StringValue(strings.ToLower(valToString(val))), nil

	case "ABS":
		if len(args) != 1 {
			return NullValue, fmt.Errorf("ABS requires exactly 1 argument")
		}
		val, err := Evaluate(args[0], ctx, opts)
		if err != nil {
			return NullValue, err
		}
		if val.Type == ValNull {
			return NullValue, nil
		}
		d, err := valueToDecimal(val)
		if err != nil {
			return NullValue, err
		}
		return DecimalValue(d.Abs()), nil

	case "CONCAT":
		var b strings.Builder
		for _, arg := range args {
			val, err := Evaluate(arg, ctx, opts)
			if err != nil {
				return NullValue, err
			}
			b.WriteString(valToString(val))
		}
		return StringValue(b.String()), nil

	case "EVENT":
		if len(args) < 2 || len(args) > 3 {
			return NullValue, fmt.Errorf("EVENT requires 2 or 3 arguments")
		}
		typeVal, err := Evaluate(args[0], ctx, opts)
		if err != nil {
			return NullValue, err
		}
		subtypeVal, err := Evaluate(args[1], ctx, opts)
		if err != nil {
			return NullValue, err
		}
		typeStr := valToString(typeVal)
		subtypeStr := valToString(subtypeVal)

		if opts == nil || opts.EventResolver == nil {
			return NullValue, fmt.Errorf("no event_resolver provided")
		}

		key := typeStr + "." + subtypeStr
		visited := opts.Visited
		if visited == nil {
			visited = make(map[string]bool)
		}
		if visited[key] {
			return NullValue, fmt.Errorf("circular reference detected: %s", key)
		}

		evalCtx := ctx
		if len(args) == 3 {
			if args[2].Kind != NodeIdentifier {
				return NullValue, fmt.Errorf("EVENT third argument must be an identifier")
			}
			if ctx.Nested == nil {
				return NullValue, fmt.Errorf("nested context '%s' not found", args[2].StrVal)
			}
			nested, ok := ctx.Nested[args[2].StrVal]
			if !ok {
				return NullValue, fmt.Errorf("nested context '%s' not found", args[2].StrVal)
			}
			evalCtx = nested
		}

		return opts.EventResolver(typeStr, subtypeStr, evalCtx, visited)
	}

	return NullValue, fmt.Errorf("unknown function: %s", name)
}
