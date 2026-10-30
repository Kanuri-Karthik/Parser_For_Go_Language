# Go Language Grammar Specification

**Project:** Robust Go Parser & AST Engine  
**Implementation:** Recursive-Descent Parser + Pratt Expression Engine + Panic-Mode Recovery  
**Specification Standard:** ISO/IEC 14977 & The Go Programming Language Specification  
**Canonical File Location:** `d:\Downloads\CD_CBP\GRAMMAR.md` (and `grammar.ebnf`)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Lexical Grammar](#2-lexical-grammar)
   - [Character Set & Whitespace](#character-set--whitespace)
   - [Comments](#comments)
   - [Identifiers & Keywords](#identifiers--keywords)
   - [Literals](#literals)
   - [Operators & Delimiters](#operators--delimiters)
3. [Automatic Semicolon Insertion (ASI)](#3-automatic-semicolon-insertion-asi)
4. [Syntactic Grammar (EBNF)](#4-syntactic-grammar-ebnf)
   - [Source File & Packages](#source-file--packages)
   - [Import Declarations](#import-declarations)
   - [Variable Declarations](#variable-declarations)
   - [Function Declarations](#function-declarations)
   - [Blocks & Statements](#blocks--statements)
   - [Control Flow Statements](#control-flow-statements)
5. [Pratt Expression Grammar & Operator Precedence](#5-pratt-expression-grammar--operator-precedence)
   - [The 5 Binary Precedence Levels](#the-5-binary-precedence-levels)
   - [Binding Powers Table](#binding-powers-table)
   - [Associativity & Precedence Algorithm](#associativity--precedence-algorithm)
6. [Panic-Mode Error Recovery & Synchronization](#6-panic-mode-error-recovery--synchronization)
7. [Grammar to Code & AST Mapping](#7-grammar-to-code--ast-mapping)
8. [Complete Example Derivations](#8-complete-example-derivations)

---

## 1. Overview

This project implements a production-grade, hand-written compiler frontend for a major subset of the Go programming language without any external dependencies.

The grammar is structured into two complementary layers:
1. **Recursive-Descent Grammar**: Handles declarative constructs, packages, imports, functions, code blocks, and control-flow statements (`parser.go`).
2. **Pratt Operator-Precedence Grammar**: Handles complex expressions, parenthesized groups, unary prefix operators, function calls, slice indexers, selectors, and binary arithmetic/logical operators with proper binding powers (`pratt.go`).

```
                ┌───────────────────────────────────┐
                │        Source Code (Go Text)      │
                └─────────────────┬─────────────────┘
                                  │ UTF-8 Scanner / ASI
                                  ▼
                ┌───────────────────────────────────┐
                │          Token Stream             │
                │ (with automatic semicolons)       │
                └─────────────────┬─────────────────┘
                                  │
         ┌────────────────────────┴────────────────────────┐
         │ Recursive-Descent                               │ Pratt Precedence
         ▼                                                 ▼
┌─────────────────────────┐                       ┌─────────────────────────┐
│ Declarations & Stmts    │                       │ Expressions             │
│ (package, var, func,    │                       │ (binary, unary, call,   │
│  if, for, return, :=)   │                       │  index, selector)       │
└────────────┬────────────┘                       └────────────┬────────────┘
             │                                                 │
             └────────────────────┬────────────────────────────┘
                                  ▼
                ┌───────────────────────────────────┐
                │      Abstract Syntax Tree (AST)   │
                │        (ast.go / AST Printer)     │
                └─────────────────┬─────────────────┘
                                  │
                                  ▼
                ┌───────────────────────────────────┐
                │      AST Evaluator / Diagnostics  │
                │      (eval.go / Panic Recovery)   │
                └───────────────────────────────────┘
```

---

## 2. Lexical Grammar

### Character Set & Whitespace
Source code is encoded in UTF-8. Whitespace is defined as spaces (`U+0020`), tabs (`U+0009`), carriage returns (`\r`), and newlines (`\n`). Newlines trigger Automatic Semicolon Insertion (ASI) when preceding certain tokens.

### Comments
```ebnf
Comment      = LineComment | GeneralComment ;
LineComment  = "//" { unicode_char - "\n" } "\n" ;
GeneralComment = "/*" { unicode_char } "*/" ;
```

### Identifiers & Keywords
```ebnf
IDENT        = letter { letter | unicode_digit } ;
letter       = "A" .. "Z" | "a" .. "z" | "_" ;
unicode_digit= "0" .. "9" ;
```

**Keywords Reserved by the Grammar:**
```
package    import     var        const      type
func       if         else       for        return
break      continue
```

### Literals
```ebnf
Literal      = INT | FLOAT | STRING ;

INT          = decimal_digits ;
decimal_digits = { "0" .. "9" }+ ;

FLOAT        = decimal_digits "." [ decimal_digits ] ;

STRING       = `"` { unicode_char - (`"` | `\`) | escaped_char } `"` ;
escaped_char = `\` ( `a` | `b` | `f` | `n` | `r` | `t` | `v` | `\` | `'` | `"` ) ;
```

### Operators & Delimiters
```
+    -    *    /    %    &    |    ^    <<    >>    &^
+=   -=   *=   /=   &&   ||   !    ==   !=    <     <=    >    >=
=    :=   ++   --   (    )    [    ]    {     }     ,     .    ;    :
```

---

## 3. Automatic Semicolon Insertion (ASI)

Go's formal specification does not require explicit semicolons at the end of statements. The lexical analyzer (`lexer.go`) automatically inserts a virtual `SEMICOLON` token (`\n auto`) when a newline character `\n` or `EOF` is reached and the immediately preceding token is:

1. **An Identifier** (`IDENT`)
2. **A Literal**: integer (`INT`), floating-point (`FLOAT`), or string (`STRING`)
3. **A Jump Keyword**: `return`, `break`, or `continue`
4. **An Operator or Delimiter**: `++`, `--`, `)`, `]`, or `}`

```
Token preceding \n                  ASI Triggered?   Inserted Token
-------------------------------------------------------------------
IDENT (e.g., x, result)             YES              SEMICOLON
INT / FLOAT / STRING                YES              SEMICOLON
return, break, continue             YES              SEMICOLON
++, --                              YES              SEMICOLON
), ], }                             YES              SEMICOLON
+, -, *, /, =, :=, (, [, {, else    NO               (none)
```

---

## 4. Syntactic Grammar (EBNF)

### Source File & Packages
```ebnf
SourceFile       = [ PackageClause ] { TopLevelDecl } EOF ;
PackageClause    = "package" PackageName ( ";" | "\n" ) ;
PackageName      = IDENT ;

TopLevelDecl     = ImportDecl
                 | VarDecl
                 | FuncDecl
                 | Stmt ;
```

### Import Declarations
```ebnf
ImportDecl       = "import" ( ImportSpec | "(" { ImportSpec ( ";" | "\n" ) } ")" ) ;
ImportSpec       = [ "." | IDENT ] ImportPath ;
ImportPath       = STRING ;
```

### Variable Declarations
```ebnf
VarDecl          = "var" IdentifierList [ Type ] [ "=" ExpressionList ] ;
IdentifierList   = IDENT { "," IDENT } ;
Type             = IDENT ;
ExpressionList   = Expression { "," Expression } ;
```

*Examples:*
```go
var x int = 10
var a, b int = 1, 2
var greeting = "Hello World"
```

### Function Declarations
```ebnf
FuncDecl         = "func" FunctionName Signature [ Block ] ;
FunctionName     = IDENT ;
Signature        = Parameters [ Result ] ;
Parameters       = "(" [ ParameterList ] ")" ;
ParameterList    = ParameterDecl { "," ParameterDecl } ;
ParameterDecl    = [ IdentifierList ] Type ;
Result           = Type | "(" [ ParameterList ] ")" ;
```

*Examples:*
```go
func add(a int, b int) int {
    return a + b
}

func main() {
    fmt.Println("Done")
}
```

### Blocks & Statements
```ebnf
Block            = "{" StatementList "}" ;
StatementList    = { Stmt [ ";" | "\n" ] } ;

Stmt             = DeclarationStmt
                 | SimpleStmt
                 | IfStmt
                 | ForStmt
                 | ReturnStmt
                 | BranchStmt
                 | Block
                 | EmptyStmt ;

DeclarationStmt  = VarDecl ;
EmptyStmt        = (* empty *) ;

SimpleStmt       = ShortVarDecl
                 | Assignment
                 | IncDecStmt
                 | ExpressionStmt ;

ShortVarDecl     = Expression ":=" Expression ;
Assignment       = Expression assign_op Expression ;
assign_op        = "=" | "+=" | "-=" | "*=" | "/=" ;

IncDecStmt       = Expression ( "++" | "--" ) ;
ExpressionStmt   = Expression ;

BranchStmt       = "break" | "continue" ;
ReturnStmt       = "return" [ ExpressionList ] ;
```

### Control Flow Statements

#### If Statement
```ebnf
IfStmt           = "if" [ SimpleStmt ";" ] Expression Block [ "else" ( IfStmt | Block ) ] ;
```

*Examples:*
```go
if x > 0 {
    return x
} else {
    return -x
}
```

#### For Statement (3 forms)
```ebnf
ForStmt          = "for" [ ForClause | Expression ] Block ;
ForClause        = [ SimpleStmt ] ";" [ Expression ] ";" [ SimpleStmt ] ;
```

*Forms supported:*
1. **Traditional 3-clause loop**: `for i := 0; i < 10; i++ { ... }`
2. **While-style conditional loop**: `for n > 0 { ... }`
3. **Infinite loop**: `for { ... }`

---

## 5. Pratt Expression Grammar & Operator Precedence

Expression parsing uses **Vaughan Pratt’s Top Down Operator Precedence** algorithm (`pratt.go`).

### The 5 Binary Precedence Levels (Go Specification)

In accordance with Go's language specification, binary operators are grouped into 5 strict precedence tiers:

```ebnf
Expression       = UnaryExpr | BinaryExpr | PrimaryExpr ;

UnaryExpr        = unary_op Expression ;
unary_op         = "+" | "-" | "!" | "^" | "*" | "&" ;

BinaryExpr       = Expression binary_op Expression ;
binary_op        = rel_op | add_op | mul_op | log_or | log_and ;

log_or           = "||" ;                                (* Precedence Level 1 *)
log_and          = "&&" ;                                (* Precedence Level 2 *)
rel_op           = "==" | "!=" | "<" | "<=" | ">" | ">=" ;(* Precedence Level 3 *)
add_op           = "+" | "-" | "|" | "^" ;               (* Precedence Level 4 *)
mul_op           = "*" | "/" | "%" | "<<" | ">>" | "&" | "&^" ; (* Precedence Level 5 *)

PrimaryExpr      = Operand { Selector | Index | Arguments } ;
Selector         = "." IDENT ;
Index            = "[" Expression "]" ;
Arguments        = "(" [ ExpressionList ] ")" ;

Operand          = IDENT
                 | Literal
                 | "(" Expression ")" ;
```

### Binding Powers Table

Left associativity is formally achieved by making the **Right Binding Power (Right BP)** equal to `Left BP + 1`.

| Tier | Precedence Level | Operators | Left BP | Right BP | Associativity | Description |
|:---:|:---:|:---|:---:|:---:|:---:|:---|
| **Postfix** | 7 | `( )` (call), `[ ]` (index), `.` (selector) | **70** | — | Left | Function calls, indexing, dot selectors |
| **Prefix** | 6 | `+`, `-`, `!`, `^`, `*`, `&` | — | **60** | Right | Unary signs, boolean NOT, bitwise NOT, pointer deref/addr |
| **Binary 5** | 5 | `*`, `/`, `%`, `<<`, `>>`, `&`, `&^` | **50** | **51** | Left | Multiplication, division, modulo, shifts, bitwise AND |
| **Binary 4** | 4 | `+`, `-`, `\|`, `^` | **40** | **41** | Left | Addition, subtraction, bitwise OR, bitwise XOR |
| **Binary 3** | 3 | `==`, `!=`, `<`, `<=`, `>`, `>=` | **30** | **31** | Left | Comparison and relational equality operators |
| **Binary 2** | 2 | `&&` | **20** | **21** | Left | Logical conjunction (AND) |
| **Binary 1** | 1 | `\|\|` | **10** | **11** | Left | Logical disjunction (OR) |

### Associativity & Precedence Algorithm

The core Pratt parsing loop in `pratt.go`:

```go
func (p *Parser) parseExpr(minBP int) Expr {
    // 1. Prefix (nud - null denotation)
    left := p.parsePrefix()

    // 2. Infix & Postfix (led - left denotation)
    for {
        cur := p.cur()
        
        // Handle Postfix (call, index, selector) at BP 70
        if isPostfix(cur) && 70 >= minBP {
            left = p.parsePostfix(left)
            continue
        }

        // Handle Binary Infix
        leftBP, rightBP := binaryPrecedence(cur.Type)
        if leftBP == -1 || leftBP < minBP {
            break // Binding power of next operator is weaker; return current subtree
        }

        p.advance()
        rhs := p.parseExpr(rightBP) // Recurse with higher right binding power
        left = &BinaryExpr{Op: cur.Type, X: left, Y: rhs}
    }
    return left
}
```

#### Precedence Resolution Examples
1. `a + b * c`
   - `+` has Left BP 40, Right BP 41.
   - When parsing right-hand side of `+`, `minBP` is 41.
   - `*` has Left BP 50. Since $50 \ge 41$, `*` binds `b` and `c` first.
   - AST Tree: `+(a, *(b, c))`

2. `10 - 4 - 3`
   - First `-` has Left BP 40, Right BP 41.
   - When parsing right-hand side, second `-` has Left BP 40.
   - Since $40 < 41$, the loop terminates. The first `-` claims `4` as its right operand.
   - AST Tree: `-(-(10, 4), 3)` (Left-associative)

---

## 6. Panic-Mode Error Recovery & Synchronization

When an unexpected token is encountered, the parser does not abort. Instead, it enters **Panic Mode** and synchronizes to the next clean statement boundary.

### Recovery Invariants
1. **Diagnostic Emission**: Generates a structured error with exact line, column, expected token, found token, and a visual ASCII caret pointer (`^^^^^^`).
2. **Synchronization Rule (`synchronize()`)**:
   - Skips tokens until reaching either `;` (consumed to start fresh) or a statement/declaration boundary keyword without consuming it.
3. **Boundary Synchronization Set**:
   ```
   { package, import, var, const, type, func, if, for, return, break, continue, } }
   ```
4. **Forward Progress Guarantee**: If no tokens were consumed during recovery, the parser advances at least one token to prevent infinite loops.

---

## 7. Grammar to Code & AST Mapping

Every non-terminal in this grammar corresponds directly to an AST struct in `ast.go` and a parsing method in `parser.go` / `pratt.go`:

| Grammar Non-Terminal | AST Struct (`ast.go`) | Parser Function (`parser.go` / `pratt.go`) |
|:---|:---|:---|
| `SourceFile` | `*File` | `Parser.ParseFile()` |
| `ImportDecl` | `*ImportDecl`, `*ImportSpec` | `Parser.parseImportDecl()` |
| `VarDecl` | `*VarDecl` | `Parser.parseVarDecl()` |
| `FuncDecl` | `*FuncDecl` | `Parser.parseFuncDecl()` |
| `Block` | `*BlockStmt` | `Parser.parseBlock()` |
| `Stmt` | `Stmt` (Interface) | `Parser.parseStmt()` |
| `ShortVarDecl`, `Assignment` | `*AssignStmt` | `Parser.parseStmt()` |
| `IncDecStmt` | `*IncDecStmt` | `Parser.parseStmt()` |
| `IfStmt` | `*IfStmt` | `Parser.parseIfStmt()` |
| `ForStmt` | `*ForStmt` | `Parser.parseForStmt()` |
| `ReturnStmt` | `*ReturnStmt` | `Parser.parseStmt()` |
| `BinaryExpr` | `*BinaryExpr` | `Parser.parseExpr(minBP)` |
| `UnaryExpr` | `*UnaryExpr` | `Parser.parseExpr(minBP)` |
| `PrimaryExpr` (Call) | `*CallExpr` | `Parser.parseExpr(minBP)` |
| `PrimaryExpr` (Index) | `*IndexExpr` | `Parser.parseExpr(minBP)` |
| `PrimaryExpr` (Selector) | `*SelectorExpr` | `Parser.parseExpr(minBP)` |
| `Operand` (Literal) | `*BasicLit` | `Parser.parseExpr(minBP)` |
| `Operand` (Ident) | `*Ident` | `Parser.parseExpr(minBP)` |

---

## 8. Complete Example Derivations

### Example 1: Function Declaration with Control Flow & Arithmetic
```go
package main

func fib(n int) int {
    if n <= 1 {
        return n
    }
    return fib(n-1) + fib(n-2)
}
```

#### AST Derivation Produced:
```
File [1:1]
├── Package: main [1:9]
└── Decls:
    └── FuncDecl: fib [3:1]
        ├── Params:
        │   └── Field: [3:10]
        │       ├── Names: n [3:10]
        │       └── Type: int [3:12]
        ├── Result: int [3:17]
        └── Body: BlockStmt [3:21]
            ├── IfStmt [4:5]
            │   ├── Cond: BinaryExpr <= [4:11]
            │   │   ├── X: n [4:8]
            │   │   └── Y: 1 [4:13]
            │   └── Body: BlockStmt [4:15]
            │       └── ReturnStmt [5:9]
            │           └── Results:
            │               └── n [5:16]
            └── ReturnStmt [7:5]
                └── Results:
                    └── BinaryExpr + [7:21]
                        ├── X: CallExpr [7:12]
                        │   ├── Fun: fib [7:12]
                        │   └── Args:
                        │       └── BinaryExpr - [7:17]
                        │           ├── X: n [7:16]
                        │           └── Y: 1 [7:18]
                        └── Y: CallExpr [7:24]
                            ├── Fun: fib [7:24]
                            └── Args:
                                └── BinaryExpr - [7:29]
                                    ├── X: n [7:28]
                                    └── Y: 2 [7:30]
```

---

## 9. Quick Reference Summary

- **Grammar Specification File**: `d:\Downloads\CD_CBP\GRAMMAR.md`
- **Pure EBNF Production File**: `d:\Downloads\CD_CBP\grammar.ebnf`
- **Lexical Scanner & ASI**: `d:\Downloads\CD_CBP\lexer.go`
- **Pratt Precedence Engine**: `d:\Downloads\CD_CBP\pratt.go`
- **Recursive-Descent Parser**: `d:\Downloads\CD_CBP\parser.go`
- **AST Node Definitions & Visualizer**: `d:\Downloads\CD_CBP\ast.go`
- **Web Playground Server**: Run with `.\parser.exe -serve :8080` or `go run . -serve :8080` and navigate to `http://localhost:8080`.
