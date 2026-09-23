package dsqlex

import "testing"

func TestLexerOperators(t *testing.T) {
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

func TestLexerFunctionKeywords(t *testing.T) {
	tokens, err := Tokenize("ROUND COALESCE NVL UPPER LOWER ABS CONCAT EVENT")
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Type != TokFnCoalesce || tokens[2].Type != TokFnCoalesce {
		t.Fatal("NVL should map to COALESCE")
	}
}

func TestLexerLineComment(t *testing.T) {
	tokens, _ := Tokenize("amount -- comment\n+ rate")
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
}

func TestLexerHashComment(t *testing.T) {
	tokens, _ := Tokenize("amount # hash\n+ rate")
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
}

func TestLexerBlockComment(t *testing.T) {
	tokens, _ := Tokenize("amount /* block */ + rate")
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
}

func TestLexerUnterminatedString(t *testing.T) {
	_, err := Tokenize("'unterminated")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLexerUnterminatedBlockComment(t *testing.T) {
	_, err := Tokenize("/* unterminated")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLexerUnexpectedCharacter(t *testing.T) {
	_, err := Tokenize("amount @ rate")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLexerStructuralTokens(t *testing.T) {
	tokens, _ := Tokenize("( ) ,")
	if tokens[0].Type != TokLParen || tokens[1].Type != TokRParen || tokens[2].Type != TokComma {
		t.Fatal("structural tokens incorrect")
	}
}

func TestLexerIsInLikeNot(t *testing.T) {
	tokens, _ := Tokenize("IS IN LIKE NOT")
	if tokens[0].Type != TokIs || tokens[1].Type != TokIn || tokens[2].Type != TokLike || tokens[3].Type != TokNot {
		t.Fatal("IS/IN/LIKE/NOT tokens incorrect")
	}
}

func TestLexerLeastGreatestTokens(t *testing.T) {
	tokens, err := Tokenize("LEAST GREATEST least Greatest")
	if err != nil {
		t.Fatal(err)
	}
	expected := []TokenType{TokFnLeast, TokFnGreatest, TokFnLeast, TokFnGreatest}
	for i, e := range expected {
		if tokens[i].Type != e {
			t.Fatalf("token %d: expected %d, got %d", i, e, tokens[i].Type)
		}
	}
}

func TestLexerTrailingQuestionMark(t *testing.T) {
	tokens, err := Tokenize("active?")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokIdentifier || tokens[0].Text != "active?" {
		t.Fatal("expected identifier 'active?'")
	}
}

func TestLexerTrailingQuestionMarkDotted(t *testing.T) {
	tokens, err := Tokenize("user.active?")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokIdentifier || tokens[0].Text != "user.active?" {
		t.Fatal("expected identifier 'user.active?'")
	}
}

func TestLexerQuestionMarkNotKeyword(t *testing.T) {
	tokens, err := Tokenize("select?")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokIdentifier || tokens[0].Text != "select?" {
		t.Fatal("expected identifier 'select?'")
	}
}

func TestLexerDoubleQuestionMarkRejected(t *testing.T) {
	if _, err := Tokenize("a??"); err == nil {
		t.Fatal("expected error for 'a??'")
	}
}

func TestLexerMinusTokenUnchanged(t *testing.T) {
	tokens, err := Tokenize("a - b")
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 3 || tokens[1].Type != TokMinus {
		t.Fatal("minus token incorrect")
	}
}
