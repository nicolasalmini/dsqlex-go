package dsqlex

// Parse tokenizes and parses an expression into an AST.
func Parse(expression string) (*AstNode, error) {
	tokens, err := Tokenize(expression)
	if err != nil {
		return nil, err
	}
	return parseTokens(tokens)
}

// Eval evaluates a pre-parsed AST with the given context.
func Eval(ast *AstNode, ctx *Context) (Value, error) {
	return Evaluate(ast, ctx, nil)
}

// EvalWithOptions evaluates a pre-parsed AST with options.
func EvalWithOptions(ast *AstNode, ctx *Context, opts *EvalOptions) (Value, error) {
	return Evaluate(ast, ctx, opts)
}

// EvalString parses and evaluates in one call.
func EvalString(expression string, ctx *Context) (Value, error) {
	ast, err := Parse(expression)
	if err != nil {
		return NullValue, err
	}
	return Eval(ast, ctx)
}
