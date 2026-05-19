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

// ═══════════════════════════════════════
// LEXER TESTS
// ═══════════════════════════════════════

func TestLexerSimpleTokens(t *testing.T) {
	tokens, err := Tokenize("+ - * / = != < > <= >=")
	if err != nil {
		t.Fatal(err)
	}
	expected := []TokenType{TokPlus, TokMinus, TokMultiply, TokDivide, TokEq, TokNeq, TokLt, TokGt, TokLte, TokGte}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, e := range expected {
		if tokens[i].Type != e {
			t.Fatalf("token %d: expected %d, got %d", i, e, tokens[i].Type)
		}
	}
}

func TestLexerKeywords(t *testing.T) {
	tokens, err := Tokenize("SELECT CASE WHEN THEN ELSE END AND OR NULL TRUE FALSE")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 11 {
		t.Fatalf("expected 11 tokens, got %d", len(tokens))
	}
}

func TestLexerCaseInsensitive(t *testing.T) {
	tokens, err := Tokenize("select Case WHEN true false null")
	if err != nil {
		t.Fatal(err)
	}
	if tokens[0].Type != TokSelect || tokens[1].Type != TokCase || tokens[2].Type != TokWhen {
		t.Fatal("case insensitive keywords not recognized")
	}
}

func TestLexerNumbers(t *testing.T) {
	tokens, err := Tokenize("42 3.14 100.00")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 3 || tokens[0].Text != "42" || tokens[1].Text != "3.14" || tokens[2].Text != "100.00" {
		t.Fatal("number tokens incorrect")
	}
}

func TestLexerStrings(t *testing.T) {
	tokens, err := Tokenize("'hello' 'world' ''")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 3 || tokens[0].Text != "hello" || tokens[1].Text != "world" || tokens[2].Text != "" {
		t.Fatal("string tokens incorrect")
	}
}

func TestLexerIdentifiers(t *testing.T) {
	tokens, err := Tokenize("amount currency_rate config.pricing.margin")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 3 || tokens[2].Text != "config.pricing.margin" {
		t.Fatal("identifier tokens incorrect")
	}
}

func TestLexerFunctions(t *testing.T) {
	tokens, err := Tokenize("ROUND COALESCE NVL UPPER LOWER ABS CONCAT EVENT")
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Type != TokFnCoalesce || tokens[2].Type != TokFnCoalesce {
		t.Fatal("NVL should map to COALESCE")
	}
}

func TestLexerComments(t *testing.T) {
	t1, _ := Tokenize("amount -- comment\n+ rate")
	if len(t1) != 3 {
		t.Fatalf("line comment: expected 3 tokens, got %d", len(t1))
	}
	t2, _ := Tokenize("amount # hash\n+ rate")
	if len(t2) != 3 {
		t.Fatalf("hash comment: expected 3 tokens, got %d", len(t2))
	}
	t3, _ := Tokenize("amount /* block */ + rate")
	if len(t3) != 3 {
		t.Fatalf("block comment: expected 3 tokens, got %d", len(t3))
	}
}

func TestLexerUnterminatedString(t *testing.T) {
	_, err := Tokenize("'unterminated")
	if err == nil {
		t.Fatal("expected error for unterminated string")
	}
}

func TestLexerUnterminatedBlock(t *testing.T) {
	_, err := Tokenize("/* unterminated")
	if err == nil {
		t.Fatal("expected error for unterminated block comment")
	}
}

// ═══════════════════════════════════════
// PARSER TESTS
// ═══════════════════════════════════════

func TestParserSimpleField(t *testing.T) {
	ast, err := Parse("field1")
	if err != nil {
		t.Fatal(err)
	}
	if ast.Kind != NodeSelect || ast.Expr.Kind != NodeIdentifier || ast.Expr.StrVal != "field1" {
		t.Fatal("expected Select(Identifier(field1))")
	}
}

func TestParserSelectOptional(t *testing.T) {
	if _, err := Parse("SELECT amount"); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("amount"); err != nil {
		t.Fatal(err)
	}
}

func TestParserMixedArithmeticRejected(t *testing.T) {
	if _, err := Parse("a + b * c"); err == nil {
		t.Fatal("expected error for mixed arithmetic")
	}
}

func TestParserParenthesizedMixed(t *testing.T) {
	if _, err := Parse("(a + b) * c"); err != nil {
		t.Fatal(err)
	}
}

func TestParserCase(t *testing.T) {
	if _, err := Parse("CASE WHEN x = 1 THEN 'one' ELSE 'other' END"); err != nil {
		t.Fatal(err)
	}
}

func TestParserMixedLogicalRejected(t *testing.T) {
	if _, err := Parse("a = 1 AND b = 2 OR c = 3"); err == nil {
		t.Fatal("expected error for mixed AND/OR")
	}
}

func TestParserSameLogicalOk(t *testing.T) {
	if _, err := Parse("a = 1 AND b = 2 AND c = 3"); err != nil {
		t.Fatal(err)
	}
}

func TestParserComparisonNoChain(t *testing.T) {
	if _, err := Parse("a = 1 = 2"); err == nil {
		t.Fatal("expected error for chained comparison")
	}
}

// ═══════════════════════════════════════
// EVALUATOR TESTS
// ═══════════════════════════════════════

func TestEvalNumberLiteral(t *testing.T) {
	v, err := EvalString("42", NewContext())
	if err != nil {
		t.Fatal(err)
	}
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

func TestEvalFieldLookup(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("amount", "500.00")
	v, _ := EvalString("amount", ctx)
	assertDecEq(t, v, "500.00")
}

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

func TestEvalComparison(t *testing.T) {
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

func TestEvalLogicalAnd(t *testing.T) {
	ctx := NewContext()
	v, _ := EvalString("TRUE AND TRUE", ctx)
	assertBoolEq(t, v, true)
	v, _ = EvalString("TRUE AND FALSE", ctx)
	assertBoolEq(t, v, false)
}

func TestEvalLogicalOr(t *testing.T) {
	ctx := NewContext()
	v, _ := EvalString("TRUE OR FALSE", ctx)
	assertBoolEq(t, v, true)
	v, _ = EvalString("FALSE OR FALSE", ctx)
	assertBoolEq(t, v, false)
}

func TestEvalCaseSimple(t *testing.T) {
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

func TestEvalRound(t *testing.T) {
	ctx := NewContext()
	v, _ := EvalString("ROUND(3.14159, 2)", ctx)
	assertDecEq(t, v, "3.14")
	v, _ = EvalString("ROUND(2.555, 2)", ctx)
	assertDecEq(t, v, "2.56")
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

func TestEvalUnknownField(t *testing.T) {
	_, err := EvalString("nonexistent", NewContext())
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestEvalComplexExpression(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("price", "100.00")
	ctx.SetDecimal("quantity", "5")
	ctx.SetDecimal("discount_rate", "0.1")
	v, _ := EvalString("(price * quantity) + (price * quantity * discount_rate)", ctx)
	assertDecEq(t, v, "550.000")
}

func TestEvalCoalesceWithField(t *testing.T) {
	ctx := NewContext()
	ctx.SetNull("base_amount")
	ctx.SetDecimal("rate", "1.5")
	v, _ := EvalString("ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1), 4)", ctx)
	assertDecEq(t, v, "0.0000")
}

// ═══════════════════════════════════════
// BENCHMARK EXPRESSIONS
// ═══════════════════════════════════════

func benchContext() *Context {
	ctx := NewContext()
	ctx.SetString("field1", "revenue")
	ctx.SetDecimal("price", "100.50")
	ctx.SetDecimal("quantity", "10")
	ctx.SetDecimal("discount_rate", "0.15")
	ctx.SetDecimal("base_amount", "5000.00")
	ctx.SetDecimal("rate", "1.25")
	ctx.SetDecimal("amount", "750.00")
	ctx.SetString("status", "active")
	ctx.SetDecimal("score", "850")
	ctx.SetString("region", "NA")
	return ctx
}

func TestBenchTier1(t *testing.T) {
	ctx := benchContext()
	v, err := EvalString("field1", ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertStrEq(t, v, "revenue")
}

func TestBenchTier2(t *testing.T) {
	ctx := benchContext()
	_, err := EvalString("(price * quantity) + (price * quantity * discount_rate)", ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBenchTier3(t *testing.T) {
	ctx := benchContext()
	_, err := EvalString("ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1), 4)", ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBenchTier4(t *testing.T) {
	ctx := benchContext()
	_, err := EvalString("CASE WHEN status = 'active' THEN ROUND(amount * 1.1, 2) ELSE ROUND(amount * 0.9, 2) END", ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBenchTier5(t *testing.T) {
	ctx := benchContext()
	_, err := EvalString("CASE WHEN score > 900 THEN ROUND(amount * 0.02, 4) WHEN score > 700 THEN ROUND(amount * 0.035, 4) WHEN score > 500 THEN ROUND(amount * 0.05, 4) WHEN score > 300 THEN ROUND(amount * 0.075, 4) ELSE ROUND(amount * 0.10, 4) END", ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBenchTier6(t *testing.T) {
	ctx := benchContext()
	_, err := EvalString("CASE WHEN (region = 'NA' AND score > 800 AND status = 'active') THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * 0.92, 4) WHEN (region = 'EU' AND score > 800 AND status = 'active') THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * 0.94, 4) WHEN (region = 'NA' AND score > 600 AND status = 'active') THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * 0.96, 4) WHEN (region = 'EU' AND score > 600 AND status = 'active') THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * 0.97, 4) WHEN (region = 'NA' AND score > 400) THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * 0.98, 4) WHEN (region = 'EU' AND score > 400) THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * 0.99, 4) WHEN status = 'suspended' THEN ROUND(COALESCE(base_amount, 0) * 0.50, 4) WHEN status = 'pending' THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * COALESCE(discount_rate, 1), 4) WHEN (score < 200 AND status != 'active') THEN 0 ELSE ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1), 4) END", ctx)
	if err != nil {
		t.Fatal(err)
	}
}
