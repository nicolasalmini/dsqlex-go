package dsqlex

import (
	"testing"

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
