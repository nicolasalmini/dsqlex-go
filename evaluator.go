package dsqlex

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/govalues/decimal"
)

// Value types
type ValueType int

const (
	ValDecimal ValueType = iota
	ValString
	ValBool
	ValNull
	ValDate
	ValDateTime
	ValTime
	ValList
	ValMap
)

type ValueList struct {
	Items []Value
}

type Value struct {
	Type    ValueType
	DecVal  decimal.Decimal
	StrVal  string
	BoolV   bool
	TimeVal time.Time
	ListVal *ValueList
	MapVal  *Context
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

func DateValue(t time.Time) Value {
	return Value{Type: ValDate, TimeVal: t}
}

func DateTimeValue(t time.Time) Value {
	return Value{Type: ValDateTime, TimeVal: t}
}

func TimeValue(t time.Time) Value {
	return Value{Type: ValTime, TimeVal: t}
}

func ListValue(items []Value) Value {
	return Value{Type: ValList, ListVal: &ValueList{Items: items}}
}

func MapValue(ctx *Context) Value {
	return Value{Type: ValMap, MapVal: ctx}
}

// Context holds variable bindings.
type Context struct {
	Fields map[string]Value
	Nested map[string]*Context
	Lists  map[string][]*Context
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

func (c *Context) SetList(key string, items []*Context) {
	if c.Lists == nil {
		c.Lists = make(map[string][]*Context)
	}
	c.Lists[key] = items
}

func (c *Context) SetDate(key string, val time.Time) {
	c.Fields[key] = DateValue(val)
}

func (c *Context) SetDateTime(key string, val time.Time) {
	c.Fields[key] = DateTimeValue(val)
}

func (c *Context) SetTime(key string, val time.Time) {
	c.Fields[key] = TimeValue(val)
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
	case ValList:
		return decimal.Decimal{}, fmt.Errorf("cannot convert list to decimal")
	case ValMap:
		return decimal.Decimal{}, fmt.Errorf("cannot convert map to decimal")
	case ValDate, ValDateTime, ValTime:
		return decimal.Decimal{}, fmt.Errorf("cannot convert temporal value to decimal")
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
	case ValDate:
		return v.TimeVal.Format("2006-01-02")
	case ValDateTime:
		return v.TimeVal.Format(time.RFC3339)
	case ValTime:
		return v.TimeVal.Format("15:04:05")
	case ValList:
		if v.ListVal == nil {
			return ""
		}
		parts := make([]string, len(v.ListVal.Items))
		for i, item := range v.ListVal.Items {
			parts[i] = valToString(item)
		}
		return strings.Join(parts, ",")
	case ValMap:
		return ""
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
		return strings.Compare(lhs.StrVal, rhs.StrVal)
	}
	if lhs.Type == ValDate && rhs.Type == ValDate {
		ly, lm, ld := lhs.TimeVal.Date()
		ry, rm, rd := rhs.TimeVal.Date()
		lt := [3]int{ly, int(lm), ld}
		rt := [3]int{ry, int(rm), rd}
		for i := 0; i < 3; i++ {
			if lt[i] < rt[i] {
				return -1
			}
			if lt[i] > rt[i] {
				return 1
			}
		}
		return 0
	}
	if lhs.Type == ValDateTime && rhs.Type == ValDateTime {
		return lhs.TimeVal.Compare(rhs.TimeVal)
	}
	if lhs.Type == ValTime && rhs.Type == ValTime {
		lh, lm, ls := lhs.TimeVal.Clock()
		rh, rm, rs := rhs.TimeVal.Clock()
		lt := [4]int{lh, lm, ls, lhs.TimeVal.Nanosecond()}
		rt := [4]int{rh, rm, rs, rhs.TimeVal.Nanosecond()}
		for i := 0; i < 4; i++ {
			if lt[i] < rt[i] {
				return -1
			}
			if lt[i] > rt[i] {
				return 1
			}
		}
		return 0
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
	if strings.IndexByte(name, '.') >= 0 {
		return resolveDotPath(strings.Split(name, "."), ctx, name, opts)
	}

	if ctx.Nested != nil {
		if nested, ok := ctx.Nested[name]; ok {
			return MapValue(nested), nil
		}
	}
	if ctx.Lists != nil {
		if list, ok := ctx.Lists[name]; ok {
			out := make([]Value, 0, len(list))
			for _, item := range list {
				out = append(out, MapValue(item))
			}
			return ListValue(out), nil
		}
	}

	if opts != nil && opts.Resolver != nil {
		if opts.Visited[name] {
			return NullValue, fmt.Errorf("circular reference detected: %s", name)
		}
		return opts.Resolver(name, opts.Visited)
	}

	return NullValue, fmt.Errorf("unknown field: %s", name)
}

func isDecimalLike(v Value) bool {
	if v.Type == ValDecimal {
		return true
	}
	if v.Type == ValString {
		_, err := decimal.Parse(v.StrVal)
		return err == nil
	}
	return false
}

func resolveDotPath(parts []string, acc any, path string, opts *EvalOptions) (Value, error) {
	if len(parts) == 0 {
		if v, ok := acc.(Value); ok {
			return v, nil
		}
		if c, ok := acc.(*Context); ok {
			return MapValue(c), nil
		}
		if list, ok := acc.([]*Context); ok {
			out := make([]Value, 0, len(list))
			for _, item := range list {
				out = append(out, MapValue(item))
			}
			return ListValue(out), nil
		}
		return NullValue, fmt.Errorf("cannot access non-value at path '%s'", path)
	}

	if list, ok := acc.([]*Context); ok {
		results := make([]Value, 0, len(list))
		for _, item := range list {
			r, err := resolveDotPath(parts, item, path, opts)
			if err != nil {
				return NullValue, err
			}
			results = append(results, r)
		}
		allNumeric := true
		for _, r := range results {
			if !isDecimalLike(r) {
				allNumeric = false
				break
			}
		}
		if !allNumeric {
			return ListValue(results), nil
		}
		sum, _ := decimal.New(0, 0)
		for _, r := range results {
			d, err := valueToDecimal(r)
			if err != nil {
				return NullValue, err
			}
			sum, err = sum.Add(d)
			if err != nil {
				return NullValue, err
			}
		}
		return DecimalValue(sum), nil
	}

	c, ok := acc.(*Context)
	if !ok {
		return NullValue, fmt.Errorf("cannot access '%s' on non-map value in path '%s'", parts[0], path)
	}
	key := parts[0]
	if v, ok := c.Fields[key]; ok {
		return resolveDotPath(parts[1:], v, path, opts)
	}
	if c.Nested != nil {
		if nested, ok := c.Nested[key]; ok {
			return resolveDotPath(parts[1:], nested, path, opts)
		}
	}
	if c.Lists != nil {
		if list, ok := c.Lists[key]; ok {
			return resolveDotPath(parts[1:], list, path, opts)
		}
	}
	return NullValue, fmt.Errorf("unknown field: %s (failed at '%s')", path, key)
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

	case NodeUnaryOp:
		val, err := Evaluate(ast.Expr, ctx, opts)
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
		return DecimalValue(d.Neg()), nil

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
		if lv.Type == ValNull || rv.Type == ValNull {
			return NullValue, nil
		}
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
		precVal, err := Evaluate(args[1], ctx, opts)
		if err != nil {
			return NullValue, err
		}
		if val.Type == ValNull || precVal.Type == ValNull {
			return NullValue, nil
		}
		d, err := valueToDecimal(val)
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

	case "LEAST", "GREATEST":
		if len(args) == 0 {
			return NullValue, fmt.Errorf("LEAST/GREATEST requires at least one argument")
		}
		vals := make([]Value, len(args))
		hasNull := false
		for i, arg := range args {
			v, err := Evaluate(arg, ctx, opts)
			if err != nil {
				return NullValue, err
			}
			if v.Type == ValNull {
				hasNull = true
			}
			vals[i] = v
		}
		if hasNull {
			return NullValue, nil
		}
		target := -1
		if name == "GREATEST" {
			target = 1
		}
		best := vals[0]
		for _, v := range vals[1:] {
			if compareValues(v, best) == target {
				best = v
			}
		}
		return best, nil

	case "EVENT":
		valid := len(args) == 2 || len(args) == 3
		if valid {
			for _, a := range args {
				if a.Kind != NodeIdentifier {
					valid = false
					break
				}
			}
		}
		if !valid {
			return NullValue, fmt.Errorf("EVENT requires 2 or 3 arguments: EVENT(type, subtype) or EVENT(type, subtype, context_source)")
		}
		typeStr := args[0].StrVal
		subtypeStr := args[1].StrVal
		if len(args) == 2 {
			return resolveEvent(typeStr, subtypeStr, ctx, opts)
		}
		source := args[2].StrVal
		if ctx.Lists != nil {
			if list, ok := ctx.Lists[source]; ok {
				sum, _ := decimal.New(0, 0)
				for _, item := range list {
					v, err := resolveEvent(typeStr, subtypeStr, item, opts)
					if err != nil {
						return NullValue, err
					}
					d, err := valueToDecimal(v)
					if err != nil {
						return NullValue, err
					}
					sum, err = sum.Add(d)
					if err != nil {
						return NullValue, err
					}
				}
				return DecimalValue(sum), nil
			}
		}
		if ctx.Nested != nil {
			if nested, ok := ctx.Nested[source]; ok {
				return resolveEvent(typeStr, subtypeStr, nested, opts)
			}
		}
		if _, ok := ctx.Fields[source]; ok {
			return NullValue, fmt.Errorf("EVENT context source '%s' must be a map or list of maps", source)
		}
		return NullValue, fmt.Errorf("EVENT context source '%s' not found in context", source)
	}

	return NullValue, fmt.Errorf("unknown function: %s", name)
}

func resolveEvent(typeStr, subtypeStr string, ctx *Context, opts *EvalOptions) (Value, error) {
	if opts == nil || opts.EventResolver == nil {
		return NullValue, fmt.Errorf("EVENT() calls require an :event_resolver option")
	}
	key := typeStr + "." + subtypeStr
	if opts.Visited[key] {
		return NullValue, fmt.Errorf("circular reference detected: %s", key)
	}
	visited := make(map[string]bool, len(opts.Visited)+1)
	for k, v := range opts.Visited {
		visited[k] = v
	}
	visited[key] = true
	return opts.EventResolver(typeStr, subtypeStr, ctx, visited)
}
