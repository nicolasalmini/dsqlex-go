package dsqlex

import (
	"fmt"
	"strings"
)

var keywords = map[string]TokenType{
	"SELECT":   TokSelect,
	"CASE":     TokCase,
	"WHEN":     TokWhen,
	"THEN":     TokThen,
	"ELSE":     TokElse,
	"END":      TokEnd,
	"AND":      TokAnd,
	"OR":       TokOr,
	"NOT":      TokNot,
	"NULL":     TokNull,
	"TRUE":     TokTrue,
	"FALSE":    TokFalse,
	"IS":       TokIs,
	"IN":       TokIn,
	"LIKE":     TokLike,
	"UPPER":    TokFnUpper,
	"LOWER":    TokFnLower,
	"ROUND":    TokFnRound,
	"COALESCE": TokFnCoalesce,
	"NVL":      TokFnCoalesce,
	"ABS":      TokFnAbs,
	"CONCAT":   TokFnConcat,
	"LEAST":    TokFnLeast,
	"GREATEST": TokFnGreatest,
	"EVENT":    TokFnEvent,
}

func Tokenize(input string) ([]Token, error) {
	var tokens []Token
	i := 0
	n := len(input)

	for i < n {
		c := input[i]

		// Skip whitespace
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}

		// Line comments: -- or #
		if c == '#' || (c == '-' && i+1 < n && input[i+1] == '-') {
			for i < n && input[i] != '\n' {
				i++
			}
			continue
		}

		// Block comments: /* ... */
		if c == '/' && i+1 < n && input[i+1] == '*' {
			i += 2
			for {
				if i+1 >= n {
					return nil, fmt.Errorf("unterminated block comment")
				}
				if input[i] == '*' && input[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}

		// Two-char operators
		if i+1 < n {
			two := input[i : i+2]
			switch two {
			case "!=":
				tokens = append(tokens, Token{TokNeq, ""})
				i += 2
				continue
			case "<=":
				tokens = append(tokens, Token{TokLte, ""})
				i += 2
				continue
			case ">=":
				tokens = append(tokens, Token{TokGte, ""})
				i += 2
				continue
			}
		}

		// Single-char operators
		switch c {
		case '+':
			tokens = append(tokens, Token{TokPlus, ""})
			i++
			continue
		case '-':
			tokens = append(tokens, Token{TokMinus, ""})
			i++
			continue
		case '*':
			tokens = append(tokens, Token{TokMultiply, ""})
			i++
			continue
		case '/':
			tokens = append(tokens, Token{TokDivide, ""})
			i++
			continue
		case '=':
			tokens = append(tokens, Token{TokEq, ""})
			i++
			continue
		case '<':
			tokens = append(tokens, Token{TokLt, ""})
			i++
			continue
		case '>':
			tokens = append(tokens, Token{TokGt, ""})
			i++
			continue
		case '(':
			tokens = append(tokens, Token{TokLParen, ""})
			i++
			continue
		case ')':
			tokens = append(tokens, Token{TokRParen, ""})
			i++
			continue
		case ',':
			tokens = append(tokens, Token{TokComma, ""})
			i++
			continue
		}

		// String literal
		if c == '\'' {
			i++
			start := i
			for i < n && input[i] != '\'' {
				i++
			}
			if i >= n {
				return nil, fmt.Errorf("unterminated string literal")
			}
			tokens = append(tokens, Token{TokString, input[start:i]})
			i++ // skip closing quote
			continue
		}

		// Number literal
		if c >= '0' && c <= '9' {
			start := i
			for i < n && ((input[i] >= '0' && input[i] <= '9') || input[i] == '.') {
				i++
			}
			tokens = append(tokens, Token{TokNumber, input[start:i]})
			continue
		}

		// Identifier or keyword
		if isIdentStart(c) {
			start := i
			for i < n && isIdentCont(input[i]) {
				i++
			}
			if i < n && input[i] == '?' {
				i++
			}
			text := input[start:i]
			if tt, ok := keywords[strings.ToUpper(text)]; ok {
				tokens = append(tokens, Token{tt, text})
			} else {
				tokens = append(tokens, Token{TokIdentifier, text})
			}
			continue
		}

		return nil, fmt.Errorf("unexpected character: '%c'", c)
	}

	return tokens, nil
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9') || c == '.'
}
