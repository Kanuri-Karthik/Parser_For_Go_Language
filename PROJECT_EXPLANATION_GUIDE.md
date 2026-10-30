# Complete Codebase Guide & Defense Script

**Project:** Robust Go Parser & AST Engine  
**Purpose:** Comprehensive guide to explain the architecture, algorithms, file structure, and design decisions to professors, evaluators, and project teammates.

---

## 1. 30-Second Elevator Pitch (Start Your Demo With This)

> *"Our project is a production-grade, hand-written compiler frontend and AST execution engine for the Go programming language, implemented in pure Go with **zero external dependencies**. Unlike standard educational parsers that crash on the first syntax error, our engine implements **panic-mode error recovery** that catches multiple errors in a single pass with visual caret diagnostics. For expressions, it uses a **Pratt operator-precedence engine** with Go's exact 5 precedence levels and binding powers. Furthermore, it incorporates an $O(n)$ single-pass scanner with **Automatic Semicolon Insertion (ASI)**, a Unicode AST visualizer, and a dynamic AST evaluator with an interactive Web Playground."*

---

## 2. End-to-End Compiler Pipeline

```
┌────────────────────────────────────────────────────────────────────────┐
│                        1. SOURCE CODE (Go Text)                        │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ Single-pass UTF-8 Scanner
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                  2. LEXER & TOKEN STREAM (lexer.go)                    │
│  - Tracks exact Line:Column positions                                  │
│  - Evaluates Automatic Semicolon Insertion (ASI) on newlines & EOF     │
│  - Emits tokens: IDENT, INT, STRING, DEFINE, keywords, etc.            │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ Token stream (with \n auto semicolons)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│               3. PARSER & PRATT ENGINE (parser.go + pratt.go)          │
│  - Declarations & Control Flow: Recursive Descent                      │
│  - Expressions: Pratt Top-Down Operator Precedence (BP 10 to 70)       │
│  - Syntax Error Recovery: Panic-Mode synchronization at statement      │
│    boundaries without aborting execution                               │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ Abstract Syntax Tree (AST)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                   4. AST CORE MODEL (ast.go)                           │
│  - Concrete AST nodes: File, FuncDecl, IfStmt, ForStmt, SwitchStmt,    │
│    StructType, CompositeLit, SliceExpr, BinaryExpr, etc.               │
│  - Pretty Printer: Unicode box-drawing characters (├──, └──, │)        │
└──────────────────┬─────────────────────────────────┬───────────────────┘
                   │                                 │
                   ▼                                 ▼
┌──────────────────────────────────────┐ ┌───────────────────────────────┐
│       5. EVALUATOR (eval.go)         │ │   6. WEB / CLI (main.go)      │
│ - Dynamic tree-walk interpreter      │ │ - CLI flags: -ast, -tokens,   │
│ - Stdin fmt.Scan(&n) & stdout        │ │   -diag, -exec, -serve        │
│ - Variables, Structs, Slices, Loops  │ │ - Web UI: Live editor, error  │
│ - Switch matching & Defer LIFO stack │ │   gutter, preset samples, EBNF│
└──────────────────────────────────────┘ └───────────────────────────────┘
```

---

## 3. File-by-File Breakdown

| File | Primary Responsibility | Key Functions / Structs | What to Highlight in Presentation |
|---|---|---|---|
| **`lexer.go`** | Lexical analysis & token generation | `Lexer`, `NextToken()`, `Tokenize()`, `keywords` map | UTF-8 rune handling, $O(n)$ single-pass scanner, and the Automatic Semicolon Insertion state machine. |
| **`ast.go`** | Abstract Syntax Tree node definitions | `Node`, `Expr`, `Stmt`, `Decl`, `File`, `FormatASTNode()` | Node structure closely modeled on Go's official `go/ast`. Formats trees with Unicode box-drawing branches (`├──`, `└──`). |
| **`pratt.go`** | Operator precedence expression parser | `parseExpr(minBP)`, `binaryPrecedence()` | Pratt parsing algorithm. 5 binary precedence levels, unary prefix (BP 60), postfix calls/indices/slices/literals (BP 70). |
| **`parser.go`** | Recursive-descent statement parser & error recovery | `Parser`, `ParseFile()`, `parseStmt()`, `synchronize()` | Panic-mode recovery: never crashes on invalid code; resynchronizes at `;` or boundary keywords and produces caret previews (`^^^^^^`). |
| **`eval.go`** | Dynamic AST runtime evaluator | `Interpreter`, `Env`, `Eval()`, `ExecStmt()`, `Value` | Tree-walk interpreter supporting dynamic input (`fmt.Scan(&n)`), structs, slicing, switches, for-range, and defer LIFO execution. |
| **`main.go`** | CLI runner & Web Playground HTTP server | `main()`, `getPlaygroundHTML()`, `/api/run`, `/download` | Unified entry point for CLI flags (`-tokens`, `-ast`, `-exec`, `-serve`) and full-featured Web Playground on port 8080. |
| **`GRAMMAR.md`** | Canonical single grammar file | Full EBNF, Precedence Table, ASI Rules, AST mapping | Standalone document consolidating the formal specification for your professor. |
| **`FULL_GO_GRAMMAR.md`** | Official complete Go language grammar | All 25 keywords, Generics, Concurrency, Channels | Complete theoretical reference of the entire Go specification. |

---

## 4. The 4 Technical Pillars (Deep Dive)

### Pillar 1: Automatic Semicolon Insertion (ASI) — `lexer.go`
- **The Concept:** In Go, programmers rarely type semicolons. The compiler's lexer automatically inserts them.
- **How It Works in Our Code:**
  The `Lexer` maintains a boolean flag `insertSemi`. When any token is scanned that can end a statement (an identifier, literal integer/string/float, `return`, `break`, `continue`, `fallthrough`, `++`, `--`, `)`, `]`, or `}`), `insertSemi` is set to `true`.
  When a newline `\n` or `EOF` is reached:
  - If `insertSemi == true`, the lexer intercepts it and yields a synthetic `SEMICOLON` token with lexeme `\n auto`.
  - If `insertSemi == false` (e.g., after `+`, `-`, `func`, `var`, `if`, `{`), no semicolon is inserted, allowing multi-line expressions like `a +\n b` to parse seamlessly.

### Pillar 2: Pratt Operator Precedence Engine — `pratt.go`
- **Why Pratt Parsing?**
  Traditional recursive-descent parsers require a separate function for every precedence level (`parseAddition`, `parseMultiplication`, `parseComparison`), leading to deep call stacks and rigid code. Vaughan Pratt's algorithm uses numerical **Binding Powers** in a single compact loop.
- **Go's 5 Precedence Levels:**
  1. Level 1 (BP 10, 11): `||` (Logical OR)
  2. Level 2 (BP 20, 21): `&&` (Logical AND)
  3. Level 3 (BP 30, 31): `==`, `!=`, `<`, `<=`, `>`, `>=` (Relational)
  4. Level 4 (BP 40, 41): `+`, `-`, `|`, `^` (Additive & bitwise)
  5. Level 5 (BP 50, 51): `*`, `/`, `%`, `<<`, `>>`, `&`, `&^` (Multiplicative & shifts)
  6. Prefix (BP 60): Unary `+`, `-`, `!`, `^`, `*`, `&`
  7. Postfix (BP 70): Calls `()`, Indices `[]`, Slices `[l:h]`, Dot selectors `.`, Composite literals `{}`
- **Associativity Rule:**
  Left associativity is guaranteed by setting `Right BP = Left BP + 1`. When parsing `10 - 4 - 3`, the first `-` has Left BP 40 and Right BP 41. When looking ahead at the second `-`, its Left BP (40) is less than the active `minBP` (41), so the loop terminates and binds `(10 - 4)` first.

### Pillar 3: Panic-Mode Error Recovery — `parser.go`
- **The Problem:** Naive parsers crash or exit on the first syntax error, making them useless in IDEs or developer tools.
- **The Solution:**
  When an expected token is missing, the parser emits a `Diagnostic` record and calls `synchronize()`.
  `synchronize()` advances through tokens until it finds a statement delimiter:
  1. A semicolon `;` (consumed to begin fresh)
  2. A statement/declaration boundary keyword (`package`, `import`, `var`, `const`, `type`, `func`, `if`, `for`, `switch`, `return`, `}`) without consuming it.
- **Caret Formatting:**
  `Diagnostic.Format()` extracts the exact source line and calculates tab indents and caret symbols:
  ```text
  error[7:7]: expected '}'
  7 |     return
    |     ^^^^^^ found RETURN
  ```
  This allows testing and reporting multiple independent syntax errors in one run (e.g. Slide 4 & 5).

### Pillar 4: Dynamic Tree-Walk Evaluator — `eval.go`
- **Execution without Compiling:**
  The `Interpreter` walks the generated AST directly.
- **Scope Management (`Env`):**
  Uses chained lexical environments (`parent *Env`). Global variables live in the root environment; functions, blocks, and loops instantiate child environments.
- **Input & Output:**
  Interprets `fmt.Scan(&n)` by reading from `env.stdin` and writing tokens back into variable memory.
- **Structs & Slices:**
  Structs evaluate to `ValStruct` (map of field names to values). Slices evaluate to `ValSlice` with sub-slicing and `append()` support.
- **Defer Mechanism:**
  Maintains a stack of deferred calls per function environment. When a function finishes execution (or returns), it pops and evaluates deferred calls in reverse order (LIFO).

---

## 5. Live Presentation / Demo Walkthrough (Step-by-Step)

When demonstrating the project live to your professor:

1. **Step 1: Open the Web Playground**
   - Navigate to `http://localhost:8080` in Chrome/Edge.
   - Point out the synchronized **Line Numbers Gutter**, **Source Editor**, **Stdin Box**, and **Tabs** (Output, AST Tree, Tokens, Diagnostics, Grammar).

2. **Step 2: Show the Presets Dropdown**
   - Select **"Sample: Scan & Math"**:
     Click **▶ Run / Parse**. Enter input `5` in Stdin. Show dynamic stdout: `5 is odd`, `Sum: 15`.
   - Switch to the **AST Tree** tab to show the Unicode tree format (`├──`, `└──`).
   - Switch to the **Tokens** tab to show the token stream with `\n auto` semicolons.

3. **Step 3: Demonstrate Phase 1 Features**
   - Select **"Sample: Structs & Fields"**:
     Click **▶ Run / Parse**. Show struct instantiation, field access, and field update.
   - Select **"Sample: Switch Statement"**:
     Show multi-case and tagless switch evaluation.
   - Select **"Sample: For-Range & Slices"**:
     Show slice sub-slicing `nums[1:4]` and `append()`.
   - Select **"Sample: Defer LIFO"**:
     Show that `Step 3` executes before `Step 2` and `Step 1`.

4. **Step 4: Demonstrate Panic-Mode Error Recovery**
   - Select **"Sample: Slide 4 Broken Code"**:
     Click **▶ Run / Parse**.
     Notice how the Line Number gutter turns red on lines with errors.
     Show the **Diagnostics** tab displaying the exact 5 errors with carets without crashing.

5. **Step 5: Show the Formal Grammar**
   - Click the **"Grammar"** tab in the web playground to show the complete EBNF productions.
   - Mention that the full repository includes standalone files [`GRAMMAR.md`](file:///d:/Downloads/CD_CBP/GRAMMAR.md) and [`FULL_GO_GRAMMAR.md`](file:///d:/Downloads/CD_CBP/FULL_GO_GRAMMAR.md).

---

## 6. Likely Professor / Viva Questions & Model Answers

### Q1: Why did you choose Pratt parsing instead of standard Recursive Descent for expressions?
> **Answer:** *"Standard recursive descent requires writing a separate recursive function for every single operator precedence tier. Go has 5 binary levels plus unary and postfix, which would result in deep call stacks, code duplication, and poor performance. Pratt parsing collapses all precedence levels into a single, elegant loop driven by numerical binding powers (BP 10 to 70), making it faster, easier to maintain, and mathematically precise."*

### Q2: How does your lexer know when to insert an automatic semicolon?
> **Answer:** *"Our lexer strictly implements the Go specification rule. It maintains an internal `insertSemi` state flag. Whenever it scans an identifier, literal (int, float, string), jump keyword (`return`, `break`, `continue`, `fallthrough`), or delimiter (`++`, `--`, `)`, `]`, `}`), it sets `insertSemi = true`. When a newline `\n` or `EOF` is reached, if `insertSemi` is true, the lexer yields a synthetic `SEMICOLON` token (`\n auto`) and resets the flag. If the preceding token was an operator like `+` or `{`, `insertSemi` is false and no semicolon is inserted."*

### Q3: What is Panic-Mode Error Recovery and how does `synchronize()` work?
> **Answer:** *"Panic mode prevents cascading compiler failures. When an expected token is missing, rather than aborting, the parser records a diagnostic and calls `synchronize()`. This skips tokens until encountering either a semicolon or a major declaration/statement keyword such as `package`, `func`, `var`, `if`, `for`, `switch`, or `}`. Once at this synchronization barrier, the parser resumes normal parsing, allowing it to discover multiple independent errors in a single compilation pass."*

### Q4: How does your dynamic evaluator execute the code without compiling to machine code?
> **Answer:** *"Our evaluator is an AST Tree-Walk Interpreter. It visits the nodes of the concrete syntax tree recursively. It uses an environment struct (`Env`) to manage lexical scoping and variable bindings. When executing a function or block, a child environment is created that links to its parent, guaranteeing proper variable shadowing and lexical scope rules."*

### Q5: How do you handle Left-Associativity in your Pratt engine?
> **Answer:** *"We use asymmetric binding powers: the Right Binding Power is set to `Left BP + 1`. For example, addition and subtraction have Left BP 40 and Right BP 41. In `10 - 4 - 3`, when the parser looks ahead at the second `-`, its Left BP (40) is not greater than the active Right BP (41), so the recursive call returns early, binding `(10 - 4)` to the left before processing `- 3`."*
