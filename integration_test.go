package dsqlex

import "testing"

// ── Compound arithmetic ──

func TestIntegrationMultiTermArithmetic(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("price", "100.00")
	ctx.SetDecimal("quantity", "5")
	ctx.SetDecimal("discount_rate", "0.1")
	v, err := EvalString("(price * quantity) + (price * quantity * discount_rate)", ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertDecEq(t, v, "550.000")
}

// ── CASE with currency dispatch ──

func TestIntegrationCaseCurrency(t *testing.T) {
	ctx := NewContext()
	ctx.SetString("currency", "BRL")
	ctx.SetDecimal("amount_local", "500.00")
	ctx.SetDecimal("amount_usd", "100.00")
	v, _ := EvalString(
		"CASE WHEN currency = 'USD' THEN amount_usd WHEN currency = 'BRL' THEN amount_local ELSE NULL END",
		ctx,
	)
	assertDecEq(t, v, "500.00")
}

// ── Nested functions with NULL ──

func TestIntegrationCoalesceWithNullField(t *testing.T) {
	ctx := NewContext()
	ctx.SetNull("base_amount")
	ctx.SetDecimal("rate", "1.5")
	v, _ := EvalString("ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1), 4)", ctx)
	assertDecEq(t, v, "0.0000")
}

func TestIntegrationCoalesceBothPresent(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("base_amount", "5000.00")
	ctx.SetDecimal("rate", "1.25")
	v, _ := EvalString("ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1), 4)", ctx)
	assertDecEq(t, v, "6250.0000")
}

// ── CASE with ROUND ──

func TestIntegrationCaseWithRoundActive(t *testing.T) {
	ctx := NewContext()
	ctx.SetString("status", "active")
	ctx.SetDecimal("amount", "750.00")
	v, _ := EvalString(
		"CASE WHEN status = 'active' THEN ROUND(amount * 1.1, 2) ELSE ROUND(amount * 0.9, 2) END",
		ctx,
	)
	assertDecEq(t, v, "825.00")
}

func TestIntegrationCaseWithRoundElse(t *testing.T) {
	ctx := NewContext()
	ctx.SetString("status", "inactive")
	ctx.SetDecimal("amount", "750.00")
	v, _ := EvalString(
		"CASE WHEN status = 'active' THEN ROUND(amount * 1.1, 2) ELSE ROUND(amount * 0.9, 2) END",
		ctx,
	)
	assertDecEq(t, v, "675.00")
}

// ── Multi-branch CASE ──

func TestIntegrationMultiBranchCaseScoreTiers(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("amount", "750.00")
	ctx.SetDecimal("score", "850")
	v, _ := EvalString(
		"CASE WHEN score > 900 THEN ROUND(amount * 0.02, 4) "+
			"WHEN score > 700 THEN ROUND(amount * 0.035, 4) "+
			"WHEN score > 500 THEN ROUND(amount * 0.05, 4) "+
			"WHEN score > 300 THEN ROUND(amount * 0.075, 4) "+
			"ELSE ROUND(amount * 0.10, 4) END",
		ctx,
	)
	// score=850 matches > 700
	assertDecEq(t, v, "26.2500")
}

// ── Complex nested CASE ──

func TestIntegrationComplexNestedCase(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("base_amount", "5000.00")
	ctx.SetDecimal("rate", "1.25")
	ctx.SetDecimal("discount_rate", "0.15")
	ctx.SetDecimal("amount", "750.00")
	ctx.SetString("status", "active")
	ctx.SetDecimal("score", "850")
	ctx.SetString("region", "NA")

	v, err := EvalString(
		"CASE WHEN (region = 'NA' AND score > 800 AND status = 'active') "+
			"THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * 0.92, 4) "+
			"WHEN (region = 'EU' AND score > 800 AND status = 'active') "+
			"THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * 0.94, 4) "+
			"WHEN (region = 'NA' AND score > 600 AND status = 'active') "+
			"THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * 0.96, 4) "+
			"WHEN (region = 'EU' AND score > 600 AND status = 'active') "+
			"THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * 0.97, 4) "+
			"WHEN (region = 'NA' AND score > 400) "+
			"THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * 0.98, 4) "+
			"WHEN (region = 'EU' AND score > 400) "+
			"THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * 0.99, 4) "+
			"WHEN status = 'suspended' THEN ROUND(COALESCE(base_amount, 0) * 0.50, 4) "+
			"WHEN status = 'pending' "+
			"THEN ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1) * COALESCE(discount_rate, 1), 4) "+
			"WHEN (score < 200 AND status != 'active') THEN 0 "+
			"ELSE ROUND(COALESCE(base_amount, 0) * COALESCE(rate, 1), 4) END",
		ctx,
	)
	if err != nil {
		t.Fatal(err)
	}
	// region=NA, score=850>800, active → first branch: ROUND(5000*1.25*0.92, 4)
	assertDecEq(t, v, "5750.0000")
}

// ── Parse-once, eval-many ──

func TestIntegrationParseOnceEvalMany(t *testing.T) {
	ast, _ := Parse("amount * rate")

	ctx1 := NewContext()
	ctx1.SetDecimal("amount", "100")
	ctx1.SetDecimal("rate", "1.5")

	ctx2 := NewContext()
	ctx2.SetDecimal("amount", "200")
	ctx2.SetDecimal("rate", "2.0")

	v1, _ := Eval(ast, ctx1)
	assertDecEq(t, v1, "150.0")
	v2, _ := Eval(ast, ctx2)
	assertDecEq(t, v2, "400.0")
}

// ── Edge cases ──

func TestIntegrationCoalesceAllNull(t *testing.T) {
	v, _ := EvalString("COALESCE(NULL, NULL, NULL)", NewContext())
	assertNull(t, v)
}

func TestIntegrationCaseNoElseNoMatch(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("x", "0")
	v, _ := EvalString("CASE WHEN x > 100 THEN 'big' END", ctx)
	assertNull(t, v)
}

func TestIntegrationNestedParenthesizedArithmetic(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("a", "2")
	ctx.SetDecimal("b", "3")
	ctx.SetDecimal("c", "4")
	v, _ := EvalString("(a + b) * (c + a)", ctx)
	// (2+3) * (4+2) = 30
	assertDecEq(t, v, "30")
}

func TestIntegrationShortCircuitAnd(t *testing.T) {
	ctx := NewContext()
	ctx.SetBool("flag", false)
	v, err := EvalString("flag AND (1 / 0 > 0)", ctx)
	if err != nil {
		t.Fatal("short-circuit AND should prevent division by zero")
	}
	assertBoolEq(t, v, false)
}

func TestIntegrationShortCircuitOr(t *testing.T) {
	ctx := NewContext()
	ctx.SetBool("flag", true)
	v, err := EvalString("flag OR (1 / 0 > 0)", ctx)
	if err != nil {
		t.Fatal("short-circuit OR should prevent division by zero")
	}
	assertBoolEq(t, v, true)
}

func TestIntegrationUnaryMinus(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("price", "500.00")
	ctx.SetNull("bonus")

	v, err := EvalString("SELECT -2.5", ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertDecEq(t, v, "-2.5")

	v, _ = EvalString("SELECT -price", ctx)
	assertDecEq(t, v, "-500.00")

	v, _ = EvalString("SELECT price * -1", ctx)
	assertDecEq(t, v, "-500.00")

	v, _ = EvalString("SELECT 5 - - 2", ctx)
	assertDecEq(t, v, "7")

	ctx2 := NewContext()
	ctx2.SetDecimal("balance", "-42")
	v, _ = EvalString("balance IN (-42, 0)", ctx2)
	assertBoolEq(t, v, true)
	v, _ = EvalString("balance IN (-41, 0)", ctx2)
	assertBoolEq(t, v, false)

	v, _ = EvalString("SELECT -bonus", ctx)
	assertNull(t, v)
}

func TestIntegrationLeastGreatestNull(t *testing.T) {
	ctx := NewContext()
	ctx.SetDecimal("price", "500.00")
	ctx.SetDecimal("quantity", "100.00")
	ctx.SetDecimal("rate", "5.00")
	ctx.SetNull("bonus")

	v, err := EvalString("SELECT LEAST(price, quantity, rate)", ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertDecEq(t, v, "5.00")

	v, _ = EvalString("SELECT GREATEST(price, quantity, rate)", ctx)
	assertDecEq(t, v, "500.00")

	v, _ = EvalString("SELECT LEAST(price, bonus)", ctx)
	assertNull(t, v)

	v, _ = EvalString("SELECT bonus + 1", ctx)
	assertNull(t, v)
	v, _ = EvalString("SELECT price * bonus", ctx)
	assertNull(t, v)

	ctx3 := NewContext()
	ctx3.SetBool("eligible?", true)
	v, _ = EvalString("SELECT eligible?", ctx3)
	assertBoolEq(t, v, true)
}
