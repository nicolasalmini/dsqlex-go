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
