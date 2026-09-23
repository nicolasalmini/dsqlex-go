package dsqlex

import "github.com/govalues/decimal"

type BinOp int

const (
	OpPlus BinOp = iota
	OpMinus
	OpMultiply
	OpDivide
	OpEq
	OpNeq
	OpLt
	OpGt
	OpLte
	OpGte
	OpAnd
	OpOr
)

type NodeKind int

const (
	NodeSelect NodeKind = iota
	NodeNumberLit
	NodeStringLit
	NodeBoolLit
	NodeNullLit
	NodeIdentifier
	NodeBinaryOp
	NodeCaseExpr
	NodeFunctionCall
	NodeInExpr
	NodeNotInExpr
	NodeLikeExpr
	NodeNotLikeExpr
	NodeUnaryOp
)

type WhenClause struct {
	Condition *AstNode
	Result    *AstNode
}

type AstNode struct {
	Kind NodeKind

	// NumberLit — pre-parsed at parse time
	DecVal decimal.Decimal

	// StringLit, Identifier, FunctionCall name
	StrVal string

	// BoolLit
	BoolVal bool

	// BinaryOp
	Op    BinOp
	Left  *AstNode
	Right *AstNode

	// CaseExpr
	Whens      []WhenClause
	ElseClause *AstNode

	// FunctionCall args, InExpr/NotInExpr items
	Args []*AstNode

	// InExpr/NotInExpr/LikeExpr/NotLikeExpr subject, Select inner
	Expr *AstNode

	// LikeExpr/NotLikeExpr pattern
	Pattern *AstNode
}
