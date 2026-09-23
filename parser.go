package dsqlex

import (
	"fmt"

	"github.com/govalues/decimal"
)

type parser struct {
	tokens []Token
	pos    int
}

func newParser(tokens []Token) *parser {
	return &parser{tokens: tokens}
}

func (p *parser) peek() *Token {
	if p.pos < len(p.tokens) {
		return &p.tokens[p.pos]
	}
	return nil
}

func (p *parser) peekType() TokenType {
	if t := p.peek(); t != nil {
		return t.Type
	}
	return -1
}

func (p *parser) advance() *Token {
	if p.pos < len(p.tokens) {
		t := &p.tokens[p.pos]
		p.pos++
		return t
	}
	return nil
}

func (p *parser) expect(ty TokenType) (*Token, error) {
	t := p.advance()
	if t == nil {
		return nil, fmt.Errorf("expected token type %d, got end of input", ty)
	}
	if t.Type != ty {
		return nil, fmt.Errorf("expected token type %d, got %d", ty, t.Type)
	}
	return t, nil
}

func (p *parser) atEnd() bool {
	return p.pos >= len(p.tokens)
}

func parseTokens(tokens []Token) (*AstNode, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty expression")
	}
	p := newParser(tokens)
	return p.parseProgram()
}

func (p *parser) parseProgram() (*AstNode, error) {
	if p.peekType() == TokSelect {
		p.advance()
	}
	expr, err := p.parseLogical()
	if err != nil {
		return nil, err
	}
	if !p.atEnd() {
		return nil, fmt.Errorf("unexpected token after expression: %d", p.peek().Type)
	}
	return &AstNode{Kind: NodeSelect, Expr: expr}, nil
}

func (p *parser) parseLogical() (*AstNode, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	tt := p.peekType()
	if tt != TokAnd && tt != TokOr {
		return left, nil
	}

	expectedTT := tt
	for p.peekType() == expectedTT {
		p.advance()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		op := OpAnd
		if expectedTT == TokOr {
			op = OpOr
		}
		left = &AstNode{Kind: NodeBinaryOp, Op: op, Left: left, Right: right}

		// Check for mixed logical operators
		next := p.peekType()
		if (next == TokAnd && expectedTT == TokOr) || (next == TokOr && expectedTT == TokAnd) {
			return nil, fmt.Errorf("ambiguous expression: mix of AND and OR requires parentheses")
		}
	}

	return left, nil
}

func (p *parser) parseComparison() (*AstNode, error) {
	left, err := p.parseArithmetic()
	if err != nil {
		return nil, err
	}

	switch p.peekType() {
	case TokEq, TokNeq, TokLt, TokGt, TokLte, TokGte:
		opTok := p.advance()
		right, err := p.parseArithmetic()
		if err != nil {
			return nil, err
		}
		var op BinOp
		switch opTok.Type {
		case TokEq:
			op = OpEq
		case TokNeq:
			op = OpNeq
		case TokLt:
			op = OpLt
		case TokGt:
			op = OpGt
		case TokLte:
			op = OpLte
		case TokGte:
			op = OpGte
		}
		// No chaining
		switch p.peekType() {
		case TokEq, TokNeq, TokLt, TokGt, TokLte, TokGte:
			return nil, fmt.Errorf("cannot chain comparison operators")
		}
		return &AstNode{Kind: NodeBinaryOp, Op: op, Left: left, Right: right}, nil

	case TokIs:
		p.advance()
		negated := false
		if p.peekType() == TokNot {
			p.advance()
			negated = true
		}
		var right *AstNode
		switch p.peekType() {
		case TokNull:
			p.advance()
			right = &AstNode{Kind: NodeNullLit}
		case TokTrue:
			p.advance()
			right = &AstNode{Kind: NodeBoolLit, BoolVal: true}
		case TokFalse:
			p.advance()
			right = &AstNode{Kind: NodeBoolLit}
		default:
			return nil, fmt.Errorf("expected NULL, TRUE, or FALSE after IS [NOT]")
		}
		op := OpEq
		if negated {
			op = OpNeq
		}
		return &AstNode{Kind: NodeBinaryOp, Op: op, Left: left, Right: right}, nil

	case TokNot:
		p.advance()
		switch p.peekType() {
		case TokIn:
			p.advance()
			items, err := p.parseInList()
			if err != nil {
				return nil, err
			}
			return &AstNode{Kind: NodeNotInExpr, Expr: left, Args: items}, nil
		case TokLike:
			p.advance()
			pattern, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			return &AstNode{Kind: NodeNotLikeExpr, Expr: left, Pattern: pattern}, nil
		default:
			return nil, fmt.Errorf("expected IN or LIKE after NOT")
		}

	case TokIn:
		p.advance()
		items, err := p.parseInList()
		if err != nil {
			return nil, err
		}
		return &AstNode{Kind: NodeInExpr, Expr: left, Args: items}, nil

	case TokLike:
		p.advance()
		pattern, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &AstNode{Kind: NodeLikeExpr, Expr: left, Pattern: pattern}, nil
	}

	return left, nil
}

func (p *parser) parseInList() ([]*AstNode, error) {
	if _, err := p.expect(TokLParen); err != nil {
		return nil, err
	}
	var items []*AstNode
	for p.peekType() != TokRParen {
		item, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		if p.peekType() == TokComma {
			p.advance()
		} else {
			break
		}
	}
	if _, err := p.expect(TokRParen); err != nil {
		return nil, err
	}
	return items, nil
}

func (p *parser) parseArithmetic() (*AstNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	group := arithGroup(p.peekType())
	if group == 0 {
		return left, nil
	}

	expectedGroup := group
	for {
		g := arithGroup(p.peekType())
		if g == 0 {
			break
		}
		if g != expectedGroup {
			return nil, fmt.Errorf("ambiguous expression: mix of +/- and */÷ requires parentheses")
		}
		opTok := p.advance()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		var op BinOp
		switch opTok.Type {
		case TokPlus:
			op = OpPlus
		case TokMinus:
			op = OpMinus
		case TokMultiply:
			op = OpMultiply
		case TokDivide:
			op = OpDivide
		}
		left = &AstNode{Kind: NodeBinaryOp, Op: op, Left: left, Right: right}
	}

	return left, nil
}

func arithGroup(tt TokenType) int {
	switch tt {
	case TokPlus, TokMinus:
		return 1 // additive
	case TokMultiply, TokDivide:
		return 2 // multiplicative
	}
	return 0
}

func (p *parser) parsePrimary() (*AstNode, error) {
	switch p.peekType() {
	case TokMinus:
		p.advance()
		operand, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &AstNode{Kind: NodeUnaryOp, Op: OpMinus, Expr: operand}, nil

	case TokNumber:
		tok := p.advance()
		d, err := decimal.Parse(tok.Text)
		if err != nil {
			return nil, fmt.Errorf("invalid number '%s': %w", tok.Text, err)
		}
		return &AstNode{Kind: NodeNumberLit, DecVal: d}, nil

	case TokString:
		tok := p.advance()
		return &AstNode{Kind: NodeStringLit, StrVal: tok.Text}, nil

	case TokTrue:
		p.advance()
		return &AstNode{Kind: NodeBoolLit, BoolVal: true}, nil

	case TokFalse:
		p.advance()
		return &AstNode{Kind: NodeBoolLit}, nil

	case TokNull:
		p.advance()
		return &AstNode{Kind: NodeNullLit}, nil

	case TokIdentifier:
		tok := p.advance()
		return &AstNode{Kind: NodeIdentifier, StrVal: tok.Text}, nil

	case TokLParen:
		p.advance()
		expr, err := p.parseLogical()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRParen); err != nil {
			return nil, err
		}
		return expr, nil

	case TokCase:
		return p.parseCase()

	case TokFnUpper, TokFnLower, TokFnRound, TokFnCoalesce, TokFnAbs, TokFnConcat, TokFnLeast, TokFnGreatest, TokFnEvent:
		return p.parseFunctionCall()

	default:
		if p.peek() != nil {
			return nil, fmt.Errorf("unexpected token: %d", p.peek().Type)
		}
		return nil, fmt.Errorf("unexpected end of input")
	}
}

func (p *parser) parseCase() (*AstNode, error) {
	if _, err := p.expect(TokCase); err != nil {
		return nil, err
	}
	var whens []WhenClause
	for p.peekType() == TokWhen {
		p.advance()
		cond, err := p.parseLogical()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokThen); err != nil {
			return nil, err
		}
		result, err := p.parseLogical()
		if err != nil {
			return nil, err
		}
		whens = append(whens, WhenClause{Condition: cond, Result: result})
	}
	if len(whens) == 0 {
		return nil, fmt.Errorf("CASE requires at least one WHEN clause")
	}
	var elseClause *AstNode
	if p.peekType() == TokElse {
		p.advance()
		var err error
		elseClause, err = p.parseLogical()
		if err != nil {
			return nil, err
		}
	}
	if _, err := p.expect(TokEnd); err != nil {
		return nil, err
	}
	return &AstNode{Kind: NodeCaseExpr, Whens: whens, ElseClause: elseClause}, nil
}

func (p *parser) parseFunctionCall() (*AstNode, error) {
	tok := p.advance()
	var name string
	switch tok.Type {
	case TokFnUpper:
		name = "UPPER"
	case TokFnLower:
		name = "LOWER"
	case TokFnRound:
		name = "ROUND"
	case TokFnCoalesce:
		name = "COALESCE"
	case TokFnAbs:
		name = "ABS"
	case TokFnConcat:
		name = "CONCAT"
	case TokFnLeast:
		name = "LEAST"
	case TokFnGreatest:
		name = "GREATEST"
	case TokFnEvent:
		name = "EVENT"
	}
	if _, err := p.expect(TokLParen); err != nil {
		return nil, err
	}
	var args []*AstNode
	if p.peekType() != TokRParen {
		first, err := p.parseLogical()
		if err != nil {
			return nil, err
		}
		args = append(args, first)
		for p.peekType() == TokComma {
			p.advance()
			arg, err := p.parseLogical()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
	}
	if _, err := p.expect(TokRParen); err != nil {
		return nil, err
	}
	return &AstNode{Kind: NodeFunctionCall, StrVal: name, Args: args}, nil
}
