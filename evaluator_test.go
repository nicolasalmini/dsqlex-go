package dsqlex

import (
	"fmt"
	"testing"
	"time"

	"github.com/govalues/decimal"
)

func mustDec(s string) decimal.Decimal {
	d, err := decimal.Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

func assertDecEq(t *testing.T, v Value, expected string) {
	t.Helper()
	if v.Type != ValDecimal {
		t.Fatalf("expected Decimal, got type %d", v.Type)
	}
	exp := mustDec(expected)
	if v.DecVal.Cmp(exp) != 0 {
		t.Fatalf("expected %s, got %s", expected, v.DecVal.String())
	}
}

func assertStrEq(t *testing.T, v Value, expected string) {
	t.Helper()
	if v.Type != ValString {
		t.Fatalf("expected String, got type %d", v.Type)
	}
	if v.StrVal != expected {
		t.Fatalf("expected %q, got %q", expected, v.StrVal)
	}
}

func assertBoolEq(t *testing.T, v Value, expected bool) {
	t.Helper()
	if v.Type != ValBool {
		t.Fatalf("expected Bool, got type %d", v.Type)
	}
	if v.BoolV != expected {
		t.Fatalf("expected %v, got %v", expected, v.BoolV)
	}
}

func assertNull(t *testing.T, v Value) {
	t.Helper()
	if v.Type != ValNull {
		t.Fatalf("expected Null, got type %d", v.Type)
	}
}

// ── Literals ──

func TestEvalNumberLiteral(t *testing.T) {
	v, _ := EvalString("42", NewContext())
	assertDecEq(t, v, "42")
}

func TestEvalStringLiteral(t *testing.T) {
	v, _ := EvalString("'hello'", NewContext())
	assertStrEq(t, v, "hello")
}

func TestEvalBoolLiterals(t *testing.T) {
	v1, _ := EvalString("TRUE", NewContext())
	assertBoolEq(t, v1, true)
	v2, _ := EvalString("FALSE", NewContext())
	assertBoolEq(t, v2, false)
}

func TestEvalNullLiteral(t *testing.T) {
	v, _ := EvalString("NULL", NewContext())
	assertNull(t, v)
}

// ── Identifiers ──

func TestEvalFieldLookup(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("amount", "500.00")
	v, _ := EvalString("amount", ctx)
	assertDecEq(t, v, "500.00")
}

func TestEvalUnknownField(t *testing.T) {
	_, err := EvalString("nonexistent", NewContext())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEvalDotPath(t *testing.T) {
	ctx := NewContext()
	nested := NewContext()
	nested.SetDecimal("rate", "5.00")
	ctx.SetNested("config", nested)
	v, _ := EvalString("config.rate", ctx)
	assertDecEq(t, v, "5.00")
}

func TestEvalNestedDotPath(t *testing.T) {
	ctx := NewContext()
	pricing := NewContext()
	pricing.SetDecimal("margin", "0.15")
	config := NewContext()
	config.SetNested("pricing", pricing)
	ctx.SetNested("config", config)
	v, _ := EvalString("config.pricing.margin", ctx)
	assertDecEq(t, v, "0.15")
}

// ── Arithmetic ──

func TestEvalArithmetic(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("a", "10")
	ctx.SetDecimal("b", "3")
	v1, _ := EvalString("a + b", ctx)
	assertDecEq(t, v1, "13")
	v2, _ := EvalString("a - b", ctx)
	assertDecEq(t, v2, "7")
	v3, _ := EvalString("a * b", ctx)
	assertDecEq(t, v3, "30")
}

func TestEvalDivision(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("a", "10")
	ctx.SetDecimal("b", "3")
	v, _ := EvalString("a / b", ctx)
	if v.Type != ValDecimal {
		t.Fatal("expected Decimal")
	}
	three33 := mustDec("3.33")
	three34 := mustDec("3.34")
	if v.DecVal.Cmp(three33) <= 0 || v.DecVal.Cmp(three34) >= 0 {
		t.Fatalf("expected ~3.33, got %s", v.DecVal.String())
	}
}

// ── Comparison ──

func TestEvalDecimalComparison(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("a", "10")
	ctx.SetDecimal("b", "20")
	v, _ := EvalString("a = a", ctx)
	assertBoolEq(t, v, true)
	v, _ = EvalString("a != b", ctx)
	assertBoolEq(t, v, true)
	v, _ = EvalString("a < b", ctx)
	assertBoolEq(t, v, true)
	v, _ = EvalString("a > b", ctx)
	assertBoolEq(t, v, false)
}

func TestEvalStringComparison(t *testing.T) {
	ctx := NewContext()
	ctx.SetString("s", "hello")
	v, _ := EvalString("s = 'hello'", ctx)
	assertBoolEq(t, v, true)
	v, _ = EvalString("s != 'world'", ctx)
	assertBoolEq(t, v, true)
}

func TestEvalNullComparison(t *testing.T) {
	ctx := NewContext()
	ctx.SetNull("x")
	v, _ := EvalString("x IS NULL", ctx)
	assertBoolEq(t, v, true)
	v, _ = EvalString("x IS NOT NULL", ctx)
	assertBoolEq(t, v, false)
}

// ── Logical ──

func TestEvalAnd(t *testing.T) {
	ctx := NewContext()
	v, _ := EvalString("TRUE AND TRUE", ctx)
	assertBoolEq(t, v, true)
	v, _ = EvalString("TRUE AND FALSE", ctx)
	assertBoolEq(t, v, false)
}

func TestEvalOr(t *testing.T) {
	ctx := NewContext()
	v, _ := EvalString("TRUE OR FALSE", ctx)
	assertBoolEq(t, v, true)
	v, _ = EvalString("FALSE OR FALSE", ctx)
	assertBoolEq(t, v, false)
}

// ── CASE/WHEN ──

func TestEvalCaseMatching(t *testing.T) {
	ctx := NewContext()
	ctx.SetString("status", "active")
	ctx.SetDecimal("amount", "100")
	v, _ := EvalString("CASE WHEN status = 'active' THEN amount ELSE 0 END", ctx)
	assertDecEq(t, v, "100")
}

func TestEvalCaseNoMatchReturnsNull(t *testing.T) {
	ctx := NewContext()
	ctx.SetString("status", "unknown")
	v, _ := EvalString("CASE WHEN status = 'active' THEN 1 WHEN status = 'pending' THEN 2 END", ctx)
	assertNull(t, v)
}

// ── Functions ──

func TestEvalRound(t *testing.T) {
	ctx := NewContext()
	v, _ := EvalString("ROUND(3.14159, 2)", ctx)
	assertDecEq(t, v, "3.14")
	v, _ = EvalString("ROUND(2.555, 2)", ctx)
	assertDecEq(t, v, "2.56")
}

func TestEvalRoundNull(t *testing.T) {
	v, _ := EvalString("ROUND(NULL, 2)", NewContext())
	assertNull(t, v)
}

func TestEvalCoalesce(t *testing.T) {
	ctx := NewContext()
	v, _ := EvalString("COALESCE(NULL, NULL, 42)", ctx)
	assertDecEq(t, v, "42")
	v, _ = EvalString("COALESCE(NULL, 'hello')", ctx)
	assertStrEq(t, v, "hello")
	v, _ = EvalString("COALESCE(NULL, NULL)", ctx)
	assertNull(t, v)
}

func TestEvalUpperLower(t *testing.T) {
	ctx := NewContext()
	v, _ := EvalString("UPPER('hello')", ctx)
	assertStrEq(t, v, "HELLO")
	v, _ = EvalString("LOWER('HELLO')", ctx)
	assertStrEq(t, v, "hello")
}

func TestEvalUpperNull(t *testing.T) {
	v, _ := EvalString("UPPER(NULL)", NewContext())
	assertNull(t, v)
}

func TestEvalAbs(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("x", "-42.5")
	v, _ := EvalString("ABS(x)", ctx)
	assertDecEq(t, v, "42.5")
}

func TestEvalConcat(t *testing.T) {
	ctx := NewContext()
	ctx.SetString("first", "Hello")
	ctx.SetString("last", "World")
	v, _ := EvalString("CONCAT(first, ' ', last)", ctx)
	assertStrEq(t, v, "Hello World")
}

// ── IN / NOT IN ──

func TestEvalIn(t *testing.T) {
	ctx := NewContext()
	ctx.SetString("status", "active")
	v, _ := EvalString("status IN ('active', 'pending')", ctx)
	assertBoolEq(t, v, true)
	v, _ = EvalString("status IN ('deleted', 'archived')", ctx)
	assertBoolEq(t, v, false)
}

func TestEvalNotIn(t *testing.T) {
	ctx := NewContext()
	ctx.SetString("status", "active")
	v, _ := EvalString("status NOT IN ('deleted')", ctx)
	assertBoolEq(t, v, true)
}

// ── LIKE / NOT LIKE ──

func TestEvalLike(t *testing.T) {
	ctx := NewContext()
	ctx.SetString("name", "Hello World")
	v, _ := EvalString("name LIKE '%world%'", ctx)
	assertBoolEq(t, v, true)
	v, _ = EvalString("name LIKE 'hello%'", ctx)
	assertBoolEq(t, v, true)
	v, _ = EvalString("name LIKE '%xyz%'", ctx)
	assertBoolEq(t, v, false)
}

func TestEvalNotLike(t *testing.T) {
	ctx := NewContext()
	ctx.SetString("name", "Hello")
	v, _ := EvalString("name NOT LIKE '%xyz%'", ctx)
	assertBoolEq(t, v, true)
}

func TestEvalLikeNull(t *testing.T) {
	ctx := NewContext()
	ctx.SetNull("name")
	v, _ := EvalString("name LIKE '%test%'", ctx)
	assertNull(t, v)
}

func TestEvalUnaryMinus(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("x", "100.00")
	ctx.SetNull("nullable_field")

	v, err := EvalString("SELECT -5", ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertDecEq(t, v, "-5")

	v, _ = EvalString("SELECT -x", ctx)
	assertDecEq(t, v, "-100.00")

	v, _ = EvalString("SELECT -(1 + 2)", ctx)
	assertDecEq(t, v, "-3")

	v, _ = EvalString("SELECT - -5", ctx)
	assertDecEq(t, v, "5")

	v, _ = EvalString("SELECT -nullable_field", ctx)
	assertNull(t, v)

	v, _ = EvalString("SELECT -NULL", ctx)
	assertNull(t, v)

	if _, err := EvalString("SELECT -'abc'", ctx); err == nil {
		t.Fatal("expected error for unary minus on non-numeric string")
	}
}

func TestEvalArithmeticNullPropagation(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("x", "10")
	ctx.SetNull("n")

	for _, expr := range []string{"n + 1", "1 + n", "n - 1", "n * 2", "n / 2", "x * n"} {
		v, err := EvalString(expr, ctx)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		assertNull(t, v)
	}
}

func TestEvalRoundAbsNull(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("x", "3.14159")
	ctx.SetNull("n")

	v, _ := EvalString("ROUND(n, 2)", ctx)
	assertNull(t, v)
	v, _ = EvalString("ROUND(x, n)", ctx)
	assertNull(t, v)
	v, _ = EvalString("ABS(n)", ctx)
	assertNull(t, v)
}

func TestEvalLeastGreatest(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("x", "100.00")
	ctx.SetDecimal("y", "20.00")
	ctx.SetNull("n")

	v, err := EvalString("LEAST(3, 1, 2)", ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertDecEq(t, v, "1")

	v, _ = EvalString("GREATEST(3, 1, 2)", ctx)
	assertDecEq(t, v, "3")

	v, _ = EvalString("LEAST(x, y)", ctx)
	assertDecEq(t, v, "20.00")

	v, _ = EvalString("LEAST(7)", ctx)
	assertDecEq(t, v, "7")

	v, _ = EvalString("LEAST(x, n)", ctx)
	assertNull(t, v)
	v, _ = EvalString("GREATEST(1, n)", ctx)
	assertNull(t, v)

	v, _ = EvalString("LEAST('banana', 'apple', 'cherry')", ctx)
	assertStrEq(t, v, "apple")
	v, _ = EvalString("GREATEST('banana', 'apple', 'cherry')", ctx)
	assertStrEq(t, v, "cherry")

	v, _ = EvalString("LEAST(1, 1.0)", ctx)
	assertDecEq(t, v, "1")
	if v.DecVal.String() != "1" {
		t.Fatalf("expected first value on tie, got %s", v.DecVal.String())
	}

	if _, err := EvalString("LEAST()", ctx); err == nil {
		t.Fatal("expected error for LEAST() with zero args")
	}
	if _, err := EvalString("GREATEST()", ctx); err == nil {
		t.Fatal("expected error for GREATEST() with zero args")
	}
}

func TestEvalLeastGreatestNumericStrings(t *testing.T) {
	ctx := NewContext()
	v, _ := EvalString("LEAST('2', '10')", ctx)
	assertStrEq(t, v, "10")
	v, _ = EvalString("GREATEST('2', '10')", ctx)
	assertStrEq(t, v, "2")
}

func TestEvalLeastGreatestDates(t *testing.T) {
	ctx := NewContext()
	ctx.SetDate("d1", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	ctx.SetDate("d2", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	ctx.SetDateTime("dt1", time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC))
	ctx.SetDateTime("dt2", time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	ctx.SetTime("t1", time.Date(0, 1, 1, 9, 30, 0, 0, time.UTC))
	ctx.SetTime("t2", time.Date(0, 1, 1, 18, 45, 0, 0, time.UTC))

	v, err := EvalString("LEAST(d1, d2)", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != ValDate || !v.TimeVal.Equal(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected date 2024-01-15, got %v", v)
	}

	v, err = EvalString("GREATEST(dt1, dt2)", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != ValDateTime || !v.TimeVal.Equal(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected datetime 12:00, got %v", v)
	}

	v, err = EvalString("LEAST(t1, t2)", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != ValTime || v.TimeVal.Hour() != 9 {
		t.Fatalf("expected time 09:30, got %v", v)
	}
}

func TestEvalUnknownDottedField(t *testing.T) {
	ctx := NewContext()
	if _, err := EvalString("missing.field", ctx); err == nil {
		t.Fatal("expected error for unknown dotted field")
	}
}

func TestEvalResolver(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("x", "100.00")
	opts := &EvalOptions{
		Resolver: func(name string, visited map[string]bool) (Value, error) {
			if name == "external" {
				return DecimalValue(mustDec("1.5")), nil
			}
			return NullValue, fmt.Errorf("unknown field: %s", name)
		},
	}
	v, err := EvalStringWithOptions("external", ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	assertDecEq(t, v, "1.5")

	if _, err := EvalStringWithOptions("nope", ctx, opts); err == nil {
		t.Fatal("expected resolver error to surface")
	}

	optsPrecedence := &EvalOptions{
		Resolver: func(name string, visited map[string]bool) (Value, error) {
			return DecimalValue(mustDec("999")), nil
		},
	}
	v, _ = EvalStringWithOptions("x", ctx, optsPrecedence)
	assertDecEq(t, v, "100.00")
}

func TestEvalResolverCircular(t *testing.T) {
	ctx := NewContext()
	opts := &EvalOptions{}
	opts.Resolver = func(name string, visited map[string]bool) (Value, error) {
		inner := &EvalOptions{Resolver: opts.Resolver}
		inner.Visited = map[string]bool{}
		for k, v := range visited {
			inner.Visited[k] = v
		}
		inner.Visited[name] = true
		return EvalStringWithOptions(name, ctx, inner)
	}
	if _, err := EvalStringWithOptions("loop", ctx, opts); err == nil {
		t.Fatal("expected circular reference error")
	}
}

func TestEvalDotPathList(t *testing.T) {
	ctx := NewContext()
	d1 := NewContext()
	d1.SetDecimal("amt", "3")
	d2 := NewContext()
	d2.SetDecimal("amt", "4")
	ctx.SetList("items", []*Context{d1, d2})
	v, err := EvalString("items.amt", ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertDecEq(t, v, "7")
}

func TestEvalDotPathListNonNumeric(t *testing.T) {
	ctx := NewContext()
	s1 := NewContext()
	s1.SetString("name", "x")
	s2 := NewContext()
	s2.SetString("name", "y")
	ctx.SetList("items", []*Context{s1, s2})
	v, err := EvalString("items.name", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != ValList || v.ListVal == nil || len(v.ListVal.Items) != 2 ||
		v.ListVal.Items[0].StrVal != "x" || v.ListVal.Items[1].StrVal != "y" {
		t.Fatalf("expected list value [x y], got %v", v)
	}
}

func TestEvalDotPathMapValue(t *testing.T) {
	ctx := NewContext()
	order := NewContext()
	order.SetDecimal("base", "7")
	ctx.SetNested("order", order)

	v, err := EvalString("order", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != ValMap || v.MapVal != order {
		t.Fatalf("expected map value, got %v", v)
	}

	outer := NewContext()
	outer.SetNested("order", order)
	wrap := NewContext()
	wrap.SetNested("o", outer)
	v2, err := EvalString("o.order", wrap)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Type != ValMap || v2.MapVal != order {
		t.Fatalf("expected map value from dot path, got %v", v2)
	}
}

func TestEvalTimeNanos(t *testing.T) {
	ctx := NewContext()
	ctx.SetTime("h1", time.Date(2000, 1, 1, 9, 0, 0, 100, time.UTC))
	ctx.SetTime("h2", time.Date(2000, 1, 1, 9, 0, 0, 200, time.UTC))
	v, err := EvalString("GREATEST(h1, h2)", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.TimeVal.Nanosecond() != 200 {
		t.Fatalf("expected h2 (200ns) to win, got %v", v.TimeVal)
	}
	v2, err := EvalString("LEAST(h1, h2)", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v2.TimeVal.Nanosecond() != 100 {
		t.Fatalf("expected h1 (100ns) to win, got %v", v2.TimeVal)
	}
}

func TestEvalDateCivilCompare(t *testing.T) {
	ctx := NewContext()
	ctx.SetDate("d1", time.Date(2024, 3, 5, 9, 0, 0, 0, time.UTC))
	ctx.SetDate("d2", time.Date(2024, 3, 5, 23, 59, 59, 0, time.UTC))
	v, err := EvalString("LEAST(d1, d2)", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if h, m, s := v.TimeVal.Clock(); h != 9 || m != 0 || s != 0 {
		t.Fatalf("expected first on tie (same civil day), got %v", v.TimeVal)
	}

	ctx2 := NewContext()
	ctx2.SetDate("e1", time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC))
	ctx2.SetDate("e2", time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC))
	v2, err := EvalString("GREATEST(e1, e2)", ctx2)
	if err != nil {
		t.Fatal(err)
	}
	if v2.TimeVal.Day() != 12 {
		t.Fatalf("expected later date to win, got %v", v2.TimeVal)
	}
	v3, err := EvalString("LEAST(e1, e2)", ctx2)
	if err != nil {
		t.Fatal(err)
	}
	if v3.TimeVal.Day() != 10 {
		t.Fatalf("expected earlier date to win, got %v", v3.TimeVal)
	}
}

func TestEvalRoundNullMissingPrecision(t *testing.T) {
	ctx := NewContext()
	ctx.SetNull("n")
	v, _ := EvalString("ROUND(n, 2)", ctx)
	assertNull(t, v)
	v, _ = EvalString("ROUND(1.5, n)", ctx)
	assertNull(t, v)
	if _, err := EvalString("ROUND(n, missing_prec)", ctx); err == nil {
		t.Fatal("expected error for missing precision field")
	}
}

func TestEvalEventTwoArg(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("amt", "10")
	opts := &EvalOptions{
		EventResolver: func(typ, subtype string, c *Context, visited map[string]bool) (Value, error) {
			return EvalStringWithOptions("amt * 2", c, &EvalOptions{Visited: visited})
		},
	}
	v, err := EvalStringWithOptions("EVENT(a, b)", ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	assertDecEq(t, v, "20")
}

func TestEvalEventNestedSource(t *testing.T) {
	ctx := NewContext()
	sub := NewContext()
	sub.SetDecimal("amt", "5")
	ctx.SetNested("sub", sub)
	opts := &EvalOptions{
		EventResolver: func(typ, subtype string, c *Context, visited map[string]bool) (Value, error) {
			return EvalStringWithOptions("amt * 2", c, &EvalOptions{Visited: visited})
		},
	}
	v, err := EvalStringWithOptions("EVENT(a, b, sub)", ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	assertDecEq(t, v, "10")
}

func TestEvalEventListSource(t *testing.T) {
	ctx := NewContext()
	i1 := NewContext()
	i1.SetDecimal("amt", "3")
	i2 := NewContext()
	i2.SetDecimal("amt", "4")
	ctx.SetList("items", []*Context{i1, i2})
	opts := &EvalOptions{
		EventResolver: func(typ, subtype string, c *Context, visited map[string]bool) (Value, error) {
			return EvalStringWithOptions("amt * 2", c, &EvalOptions{Visited: visited})
		},
	}
	v, err := EvalStringWithOptions("EVENT(a, b, items)", ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	assertDecEq(t, v, "14")

	empty := NewContext()
	empty.SetList("items", []*Context{})
	v, err = EvalStringWithOptions("EVENT(a, b, items)", empty, opts)
	if err != nil {
		t.Fatal(err)
	}
	assertDecEq(t, v, "0")
}

func TestEvalEventErrors(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("amt", "10")
	opts := &EvalOptions{
		EventResolver: func(typ, subtype string, c *Context, visited map[string]bool) (Value, error) {
			return DecimalValue(mustDec("1")), nil
		},
	}
	if _, err := EvalStringWithOptions("EVENT(a, b, missing)", ctx, opts); err == nil {
		t.Fatal("expected error for missing source")
	}
	if _, err := EvalStringWithOptions("EVENT(a, b, amt)", ctx, opts); err == nil {
		t.Fatal("expected error for non-map source")
	}
	if _, err := EvalStringWithOptions("EVENT('a', 'b')", ctx, opts); err == nil {
		t.Fatal("expected error for non-identifier args")
	}
	if _, err := EvalStringWithOptions("EVENT(a)", ctx, opts); err == nil {
		t.Fatal("expected error for wrong arity")
	}
	if _, err := EvalStringWithOptions("EVENT(a, b)", ctx, &EvalOptions{}); err == nil {
		t.Fatal("expected error for missing event resolver")
	}
}

func TestEvalEventCircular(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("amt", "10")
	opts := &EvalOptions{}
	opts.EventResolver = func(typ, subtype string, c *Context, visited map[string]bool) (Value, error) {
		return EvalStringWithOptions("EVENT(a, b)", c, &EvalOptions{EventResolver: opts.EventResolver, Visited: visited})
	}
	if _, err := EvalStringWithOptions("EVENT(a, b)", ctx, opts); err == nil {
		t.Fatal("expected circular reference error")
	}
}
