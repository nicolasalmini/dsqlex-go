package dsqlex

type TokenType int

const (
	TokPlus TokenType = iota
	TokMinus
	TokMultiply
	TokDivide
	TokEq
	TokNeq
	TokLt
	TokGt
	TokLte
	TokGte
	TokSelect
	TokCase
	TokWhen
	TokThen
	TokElse
	TokEnd
	TokAnd
	TokOr
	TokNot
	TokNull
	TokTrue
	TokFalse
	TokIs
	TokIn
	TokLike
	TokFnUpper
	TokFnLower
	TokFnRound
	TokFnCoalesce
	TokFnAbs
	TokFnConcat
	TokFnEvent
	TokNumber
	TokString
	TokIdentifier
	TokLParen
	TokRParen
	TokComma
	TokFnLeast
	TokFnGreatest
)

type Token struct {
	Type TokenType
	Text string
}
