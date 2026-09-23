package dsqlex

import "testing"

func TestParserSimpleIdentifier(t *testing.T) {
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

func TestParserArithmeticChaining(t *testing.T) {
	if _, err := Parse("a + b + c"); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("a * b * c"); err != nil {
		t.Fatal(err)
	}
}

func TestParserMixedArithmeticRejected(t *testing.T) {
	if _, err := Parse("a + b * c"); err == nil {
		t.Fatal("expected error for mixed arithmetic")
	}
	if _, err := Parse("a * b + c"); err == nil {
		t.Fatal("expected error for mixed arithmetic")
	}
}

func TestParserParenthesizedMixed(t *testing.T) {
	if _, err := Parse("(a + b) * c"); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("a * (b + c)"); err != nil {
		t.Fatal(err)
	}
}

func TestParserCaseWhenElseEnd(t *testing.T) {
	if _, err := Parse("CASE WHEN x = 1 THEN 'one' ELSE 'other' END"); err != nil {
		t.Fatal(err)
	}
}

func TestParserCaseMultipleWhen(t *testing.T) {
	if _, err := Parse("CASE WHEN x = 1 THEN 'a' WHEN x = 2 THEN 'b' END"); err != nil {
		t.Fatal(err)
	}
}

func TestParserFunctionCall(t *testing.T) {
	if _, err := Parse("ROUND(amount, 2)"); err != nil {
		t.Fatal(err)
	}
}

func TestParserNestedFunctionCall(t *testing.T) {
	if _, err := Parse("ROUND(COALESCE(x, 0), 2)"); err != nil {
		t.Fatal(err)
	}
}

func TestParserInExpression(t *testing.T) {
	if _, err := Parse("status IN ('active', 'pending')"); err != nil {
		t.Fatal(err)
	}
}

func TestParserNotInExpression(t *testing.T) {
	if _, err := Parse("status NOT IN ('deleted')"); err != nil {
		t.Fatal(err)
	}
}

func TestParserLikeExpression(t *testing.T) {
	if _, err := Parse("name LIKE '%test%'"); err != nil {
		t.Fatal(err)
	}
}

func TestParserNotLikeExpression(t *testing.T) {
	if _, err := Parse("name NOT LIKE '%test%'"); err != nil {
		t.Fatal(err)
	}
}

func TestParserIsNull(t *testing.T) {
	if _, err := Parse("x IS NULL"); err != nil {
		t.Fatal(err)
	}
}

func TestParserIsNotNull(t *testing.T) {
	if _, err := Parse("x IS NOT NULL"); err != nil {
		t.Fatal(err)
	}
}

func TestParserIsTrueFalse(t *testing.T) {
	if _, err := Parse("x IS TRUE"); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("x IS FALSE"); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("x IS NOT TRUE"); err != nil {
		t.Fatal(err)
	}
}

func TestParserMixedLogicalRejected(t *testing.T) {
	if _, err := Parse("a = 1 AND b = 2 OR c = 3"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParserSameLogicalChaining(t *testing.T) {
	if _, err := Parse("a = 1 AND b = 2 AND c = 3"); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("a = 1 OR b = 2 OR c = 3"); err != nil {
		t.Fatal(err)
	}
}

func TestParserComparisonChainingRejected(t *testing.T) {
	if _, err := Parse("a = 1 = 2"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := Parse("a < b < c"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParserEmptyExpression(t *testing.T) {
	if _, err := Parse(""); err == nil {
		t.Fatal("expected error")
	}
}

func TestParserUnaryMinusLiteral(t *testing.T) {
	ast, err := Parse("SELECT -1")
	if err != nil {
		t.Fatal(err)
	}
	u := ast.Expr
	if u.Kind != NodeUnaryOp || u.Op != OpMinus || u.Expr.Kind != NodeNumberLit {
		t.Fatal("expected Select(UnaryOp(minus, Number(1)))")
	}
}

func TestParserUnaryMinusRightOfMultiply(t *testing.T) {
	ast, err := Parse("SELECT amount * -1")
	if err != nil {
		t.Fatal(err)
	}
	b := ast.Expr
	if b.Kind != NodeBinaryOp || b.Op != OpMultiply ||
		b.Right.Kind != NodeUnaryOp || b.Right.Expr.Kind != NodeNumberLit {
		t.Fatal("expected multiply(amount, unary(-1))")
	}
}

func TestParserUnaryMinusParenthesized(t *testing.T) {
	ast, err := Parse("SELECT -(1 + 2)")
	if err != nil {
		t.Fatal(err)
	}
	u := ast.Expr
	if u.Kind != NodeUnaryOp || u.Expr.Kind != NodeBinaryOp || u.Expr.Op != OpPlus {
		t.Fatal("expected unary(-(1 + 2))")
	}
}

func TestParserSubtractionOfNegatedOperand(t *testing.T) {
	ast, err := Parse("SELECT 5 - - 2")
	if err != nil {
		t.Fatal(err)
	}
	b := ast.Expr
	if b.Kind != NodeBinaryOp || b.Op != OpMinus ||
		b.Right.Kind != NodeUnaryOp || b.Right.Expr.Kind != NodeNumberLit {
		t.Fatal("expected minus(5, unary(-2))")
	}
}

func TestParserNestedUnaryMinus(t *testing.T) {
	ast, err := Parse("SELECT - -5")
	if err != nil {
		t.Fatal(err)
	}
	u := ast.Expr
	if u.Kind != NodeUnaryOp || u.Expr.Kind != NodeUnaryOp || u.Expr.Expr.Kind != NodeNumberLit {
		t.Fatal("expected unary(unary(5))")
	}
}

func TestParserDoubleDashIsComment(t *testing.T) {
	if _, err := Parse("SELECT --5"); err == nil {
		t.Fatal("expected error: '--5' is a comment, not unary minus")
	}
}

func TestParserUnaryMinusInList(t *testing.T) {
	ast, err := Parse("x IN (1, -2)")
	if err != nil {
		t.Fatal(err)
	}
	in := ast.Expr
	if in.Kind != NodeInExpr || len(in.Args) != 2 ||
		in.Args[1].Kind != NodeUnaryOp {
		t.Fatal("expected IN list with negated second item")
	}
}

func TestParserEmptyInList(t *testing.T) {
	ast, err := Parse("x IN ()")
	if err != nil {
		t.Fatal(err)
	}
	if ast.Expr.Kind != NodeInExpr || len(ast.Expr.Args) != 0 {
		t.Fatal("expected IN with empty item list")
	}
}

func TestParserMixedArithmeticWithNegatedOperandRejected(t *testing.T) {
	if _, err := Parse("SELECT 1 + 2 * -3"); err == nil {
		t.Fatal("expected ambiguous-expression error")
	}
}

func TestParserLeastGreatestCalls(t *testing.T) {
	ast, err := Parse("LEAST(a, 1, 2)")
	if err != nil {
		t.Fatal(err)
	}
	f := ast.Expr
	if f.Kind != NodeFunctionCall || f.StrVal != "LEAST" || len(f.Args) != 3 {
		t.Fatal("expected LEAST call with 3 args")
	}
	ast2, err := Parse("GREATEST(x, y)")
	if err != nil {
		t.Fatal(err)
	}
	if ast2.Expr.Kind != NodeFunctionCall || ast2.Expr.StrVal != "GREATEST" || len(ast2.Expr.Args) != 2 {
		t.Fatal("expected GREATEST call with 2 args")
	}
}
