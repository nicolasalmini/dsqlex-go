# dsqlex-go

A Go implementation of the DSQLEX expression evaluator. Uses [`govalues/decimal`](https://github.com/govalues/decimal) for zero-allocation, 19-digit fixed-point decimal arithmetic.

## Usage

```go
import dsqlex "github.com/nicolasalmini/dsqlex-go"

// One-shot: parse + evaluate
ctx := dsqlex.NewContext()
ctx.SetDecimal("price", "100.00")
ctx.SetDecimal("quantity", "5")
ctx.SetDecimal("tax", "50.00")

result, err := dsqlex.EvalString("(price * quantity) + tax", ctx)
// result = Value{Type: ValDecimal, DecVal: 550.00}

// Parse once, evaluate many
ast, _ := dsqlex.Parse("amount * rate")
for _, record := range records {
    result, _ := dsqlex.Eval(ast, record)
}
```

## Building

```bash
go build ./...
go test ./...
```

## Dependencies

- [`govalues/decimal`](https://github.com/govalues/decimal) — High-performance, zero-allocation decimal arithmetic (19-digit precision)

## Supported Features

| Feature | Syntax |
|---------|--------|
| Arithmetic | `+`, `-`, `*`, `/` (decimal precision) |
| Comparison | `=`, `!=`, `<`, `>`, `<=`, `>=` |
| Logical | `AND`, `OR` (same-op chaining; mixing requires parens) |
| Conditionals | `CASE WHEN ... THEN ... ELSE ... END` |
| Functions | `ROUND()`, `COALESCE()`/`NVL()`, `UPPER()`, `LOWER()`, `ABS()`, `CONCAT()`, `EVENT()` |
| Membership | `IN (...)`, `NOT IN (...)` |
| Pattern | `LIKE`, `NOT LIKE` (case-insensitive) |
| Null check | `IS NULL`, `IS NOT NULL`, `IS TRUE`, `IS FALSE` |
| Literals | Numbers, strings (`'...'`), `TRUE`, `FALSE`, `NULL` |
| Dot-paths | `config.pricing.margin` (nested context access) |
| Comments | `--`, `#`, `/* ... */` |

## API

```go
// Parse tokenizes and parses an expression into an AST.
func Parse(expression string) (*AstNode, error)

// Eval evaluates a pre-parsed AST with the given context.
func Eval(ast *AstNode, ctx *Context) (Value, error)

// EvalWithOptions evaluates with resolvers.
func EvalWithOptions(ast *AstNode, ctx *Context, opts *EvalOptions) (Value, error)

// EvalString parses and evaluates in one call.
func EvalString(expression string, ctx *Context) (Value, error)
```

### Context

```go
ctx := dsqlex.NewContext()
ctx.SetDecimal("amount", "100.50")
ctx.SetString("status", "active")
ctx.SetBool("enabled", true)
ctx.SetNull("discount")

// Nested contexts for dot-path resolution
nested := dsqlex.NewContext()
nested.SetDecimal("margin", "0.15")
ctx.SetNested("config", nested)
```

### Value

```go
type Value struct {
    Type   ValueType       // ValDecimal, ValString, ValBool, ValNull
    DecVal decimal.Decimal // govalues/decimal — zero-allocation
    StrVal string
    BoolV  bool
}
```

## Design Decisions

- **Zero-allocation decimals**: `govalues/decimal` uses 128-bit integers on the stack. No heap allocations for arithmetic.
- **Struct-based Value**: `Value` is a plain struct (not an interface), avoiding virtual dispatch and heap escapes in the hot path.
- **Pre-parsed decimals in AST**: Number literals are parsed to `decimal.Decimal` at parse time, not eval time.
- **Explicit parentheses**: `a + b * c` is rejected. Use `(a + b) * c`.

## Related

- [dsqlex-c](https://github.com/nicolasalmini/dsqlex-c) — C/C++ implementation (mpdecimal)
- [dsqlex-rs](https://github.com/nicolasalmini/dsqlex-rs) — Rust implementation (rust_decimal)
- [dsqlex-py](https://github.com/nicolasalmini/dsqlex-py) — Python implementation
- [dsqlex-ts](https://github.com/nicolasalmini/dsqlex-ts) — TypeScript implementation
- [dsqlex-bench](https://github.com/nicolasalmini/dsqlex-bench) — Cross-language benchmark suite
