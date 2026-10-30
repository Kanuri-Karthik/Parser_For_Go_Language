# Robust Go Parser & AST Engine

A robust, hand-written recursive-descent and Pratt parser for Go source code, featuring automatic semicolon insertion, panic-mode error recovery with visual caret diagnostics, and Unicode/ASCII AST tree visualization.

## Architecture

- **`lexer.go`**: Single-pass rune scanner ($O(n)$) with UTF-8 support, exact `line:col` tracking, and automatic semicolon insertion rules matching the Go specification.
- **`pratt.go`**: Pratt expression engine implementing Go's 5 binary operator precedence levels (binding powers 10–50), unary prefix operators (binding power 60), postfix calls/indices/selectors (binding power 70), and left associativity.
- **`ast.go`**: AST data structures modeled on Go's `go/ast` (`File`, `VarDecl`, `FuncDecl`, `BlockStmt`, `AssignStmt`, `IfStmt`, `ForStmt`, `ReturnStmt`, `BinaryExpr`, `UnaryExpr`, `ParenExpr`, `CallExpr`, `IndexExpr`, `SelectorExpr`, etc.) with tree formatting using box-drawing characters (`├──`, `└──`, `│`).
- **`parser.go`**: Recursive-descent statement engine with panic-mode error recovery (`synchronize()`) that skips to statement boundaries without aborting. Produces diagnostics with source code line excerpts and caret indicators.
- **`main.go`**: CLI driver with flags for dumping token streams, printing AST trees, displaying panic-mode diagnostics, and executing live demos.

---

## Grammar

The complete, formal language grammar for this Go compiler project is maintained in dedicated standalone files:
- **`GRAMMAR.md`**: Complete grammar specification, lexical tokens, Automatic Semicolon Insertion (ASI) rules, Pratt operator binding powers table, panic-mode error recovery invariants, and AST mappings.
- **`grammar.ebnf`**: Clean, standard ISO/IEC 14977 & Go-style EBNF production rules.

### Quick EBNF Summary

```ebnf
SourceFile = [ "package" IDENT ";" ] { TopLevelDecl ";" } ;
TopLevel   = ImportDecl | VarDecl | FuncDecl | Stmt ;
VarDecl    = "var" IDENT [ Type ] [ "=" Expr ] ;
FuncDecl   = "func" IDENT "(" [ Params ] ")" [ Type ] Block ;
Block      = "{" { Stmt ";" } "}" ;
Stmt       = ShortVar | Assign | IncDec | If | For | Return | Expr ;
ShortVar   = IDENT ":=" Expr ;
If         = "if" Expr Block [ "else" Block ] ;
For        = "for" [ ForClause | Expr ] Block ;
```

See [GRAMMAR.md](GRAMMAR.md) for the exhaustive specification and derivation rules.

### Precedence Levels & Binding Powers

| Level | Operators | Left BP | Right BP | Associativity |
|:-----:|:----------|:-------:|:--------:|:--------------|
| **1** | `\|\|` | 10 | 11 | Left |
| **2** | `&&` | 20 | 21 | Left |
| **3** | `==`, `!=`, `<`, `<=`, `>`, `>=` | 30 | 31 | Left |
| **4** | `+`, `-`, `\|`, `^` | 40 | 41 | Left |
| **5** | `*`, `/`, `%`, `<<`, `>>`, `&`, `&^` | 50 | 51 | Left |
| Prefix| `+`, `-`, `!`, `^`, `*`, `&` | — | 60 | Right |
| Postfix | Call `()`, Index `[]`, Selector `.` | 70 | — | Left |

---

## Usage

### Dynamic Execution (AST Evaluator)
```bash
# Execute a Go file dynamically
go run . -exec testdata/calc_run.go

# Evaluate an expression dynamically
go run . -code "2 + 3 * 4"
# Output: => Result: 14
```

### Interactive Dynamic REPL
```bash
# Launch interactive REPL
go run . -repl
# Or: .\parser.exe -repl
```

### Dynamic Web Playground
```bash
# Launch dynamic visualizer & evaluator server
go run . -serve :8080
# Open http://localhost:8080 in your browser
```

### Dump Token Stream
```bash
go run . -tokens testdata/valid.go
```

### Print AST ASCII Tree
```bash
go run . -ast testdata/valid.go
```

### Parse Inline Expression
```bash
go run . -ast -code "x := a + b*2"
go run . -ast -code "(a + b) * c"
```

### Panic-Mode Error Diagnostics
```bash
go run . -diag testdata/invalid.go
```

---

## Running Tests

```bash
# Unit, snapshot, and recovery tests
go test -v ./...

# Fuzz testing for robustness
go test -fuzz=FuzzParser -fuzztime=5s
```
