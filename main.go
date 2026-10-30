//go:build !js

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const demoValid = `package main

var limit int = 10

func calc(a, b int) int {
	x := a + b*2
	x = x - (limit / 2)
	if x > 5 {
		return x
	}
	return -x
}

func main() {
	println("Calculating calc(2, 3)...")
	res := calc(2, 3)
	println("Result is:", res)
	return res
}`

const demoInvalid = `x := ;
var y int 10
z := (1 + ;
func f( {
if x > 5 return`

func printTokenStream(tokens []Token) {
	fmt.Printf("%-11s %-9s %s\n", "Token", "Lexeme", "Line:Col")
	for _, tok := range tokens {
		lex := tok.Lexeme
		if tok.IsAutoSemi {
			lex = `\n auto`
		} else if tok.Type == EOF {
			lex = "—"
		}
		fmt.Printf("%-11s %-9s %s\n", tok.Type, lex, tok.Pos)
	}
}

func formatTokenStream(tokens []Token) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-11s %-9s %s\n", "Token", "Lexeme", "Line:Col"))
	for _, tok := range tokens {
		lex := tok.Lexeme
		if tok.IsAutoSemi {
			lex = `\n auto`
		} else if tok.Type == EOF {
			lex = "—"
		}
		sb.WriteString(fmt.Sprintf("%-11s %-9s %s\n", tok.Type, lex, tok.Pos))
	}
	return sb.String()
}

func runDemo() {
	fmt.Println("==================================================")
	fmt.Println("  ROBUST GO PARSER & AST ENGINE (PRATT PARSER)    ")
	fmt.Println("==================================================")

	fmt.Println("\n--- 1. Single Expression Demo: x := a + b*2 ---")
	singleExpr := "x := a + b*2"
	exprTokens := Tokenize(singleExpr)
	fmt.Println("\n[Token Stream]:")
	printTokenStream(exprTokens)

	exprParser := NewParser(singleExpr)
	file := exprParser.ParseFile()
	fmt.Println("\n[ASCII AST Tree]:")
	if len(file.Stmts) > 0 {
		fmt.Println(PrintAST(file.Stmts[0]))
	} else {
		fmt.Println(PrintAST(file))
	}

	fmt.Println("\n--- 2. Valid Go Program (Slide 4) ---")
	fmt.Println("[Source Code]:")
	fmt.Println(demoValid)

	validParser := NewParser(demoValid)
	validFile := validParser.ParseFile()
	if len(validParser.errors) > 0 {
		fmt.Println("\n[Diagnostics]:")
		fmt.Print(validParser.FormatDiagnostics())
	} else {
		fmt.Println("\n[AST Tree]:")
		fmt.Println(PrintAST(validFile))
	}

	fmt.Println("\n[Dynamic Execution]:")
	interp := NewInterpreter()
	retVal, stdout, err := interp.ExecuteFile(validFile)
	if err != nil {
		fmt.Printf("Runtime Error: %v\n", err)
	} else {
		if stdout != "" {
			fmt.Print(stdout)
		}
		fmt.Printf("Exit Return Value: %v\n", retVal)
	}

	fmt.Println("\n--- 3. Invalid Go Program (Panic-Mode Recovery - Slide 4 & 5) ---")
	fmt.Println("[Source Code]:")
	fmt.Println(demoInvalid)

	invalidParser := NewParser(demoInvalid)
	_ = invalidParser.ParseFile()
	fmt.Println("\n[Panic-Mode Diagnostics]:")
	fmt.Print(invalidParser.FormatDiagnostics())

	fmt.Println("==================================================")
	fmt.Println("  DEMO COMPLETED SUCCESSFULLY                     ")
	fmt.Println("==================================================")
}

func runREPL() {
	fmt.Println("==================================================")
	fmt.Println("  Robust Go Parser & Dynamic Evaluator REPL       ")
	fmt.Println("  Commands:                                       ")
	fmt.Println("    :ast      Toggle AST visualization            ")
	fmt.Println("    :tokens   Toggle Token stream display         ")
	fmt.Println("    :env      Inspect environment variables       ")
	fmt.Println("    :clear    Clear current environment           ")
	fmt.Println("    :exit     Exit REPL                           ")
	fmt.Println("==================================================")

	showAST := true
	showTokens := false
	interp := NewInterpreter()
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("go> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch line {
		case ":exit", ":quit", "exit":
			fmt.Println("Exiting REPL.")
			return
		case ":ast":
			showAST = !showAST
			fmt.Printf("AST display: %v\n", showAST)
			continue
		case ":tokens":
			showTokens = !showTokens
			fmt.Printf("Token display: %v\n", showTokens)
			continue
		case ":env":
			fmt.Println("[Environment Variables]:")
			if len(interp.GlobalEnv.vars) == 0 {
				fmt.Println("  (empty)")
			}
			for k, v := range interp.GlobalEnv.vars {
				fmt.Printf("  %s = %s\n", k, v)
			}
			continue
		case ":clear":
			interp = NewInterpreter()
			fmt.Println("Environment cleared.")
			continue
		}

		if showTokens {
			toks := Tokenize(line)
			fmt.Println("[Tokens]:")
			printTokenStream(toks)
		}

		p := NewParser(line)
		file := p.ParseFile()

		if len(p.errors) > 0 {
			fmt.Print(p.FormatDiagnostics())
			continue
		}

		if showAST {
			fmt.Println("[AST]:")
			if len(file.Decls) == 0 && len(file.Stmts) == 1 && file.Name == nil {
				if exprStmt, ok := file.Stmts[0].(*ExprStmt); ok {
					fmt.Println(PrintAST(exprStmt.X))
				} else {
					fmt.Println(PrintAST(file.Stmts[0]))
				}
			} else {
				fmt.Println(PrintAST(file))
			}
		}

		val, stdout, err := interp.ExecuteFile(file)
		if err != nil {
			if stdout != "" {
				fmt.Print(stdout)
			}
			if rErr, ok := err.(*RuntimeError); ok {
				diag := Diagnostic{
					Pos:     rErr.Pos,
					Length:  1,
					Message: "runtime error: " + rErr.Message,
				}
				fmt.Fprintln(os.Stderr, diag.Format(p.srcLines))
			} else {
				fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
			}
		} else {
			if stdout != "" {
				fmt.Print(stdout)
			}
			if val.Kind != ValNil && val.Kind != 0 {
				fmt.Printf("=> %s\n", val)
			}
		}
	}
}

type APIResponse struct {
	Tokens      string `json:"tokens"`
	AST         string `json:"ast"`
	Diagnostics string `json:"diagnostics"`
	Output      string `json:"output"`
	Result      string `json:"result"`
	HasErrors   bool   `json:"has_errors"`
	ErrorLines  []int  `json:"error_lines"`
}

func runServer(addr string) {
	http.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"POST only"}`, http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Code  string `json:"code"`
			Stdin string `json:"stdin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusBadRequest)
			return
		}

		tokens := Tokenize(req.Code)
		tokStr := formatTokenStream(tokens)

		p := NewParser(req.Code)
		file := p.ParseFile()

		diagStr := p.FormatDiagnostics()
		hasErrors := len(p.errors) > 0

		var astStr string
		if len(file.Decls) == 0 && len(file.Stmts) == 1 && file.Name == nil {
			if exprStmt, ok := file.Stmts[0].(*ExprStmt); ok {
				astStr = PrintAST(exprStmt.X)
			} else {
				astStr = PrintAST(file.Stmts[0])
			}
		} else {
			astStr = PrintAST(file)
		}

		var outStr string
		var resStr string
		var errorLines []int

		for _, err := range p.errors {
			errorLines = append(errorLines, err.Pos.Line)
		}

		if hasErrors {
			outStr = fmt.Sprintf("SyntaxError: Failed to parse Go source (%d syntax error(s) found):\n\n%s", len(p.errors), diagStr)
		} else {
			interp := NewInterpreterWithIO(strings.NewReader(req.Stdin), nil)
			val, stdout, err := interp.ExecuteFile(file)
			outStr = stdout
			if err != nil {
				hasErrors = true
				var rErr *RuntimeError
				if re, ok := err.(*RuntimeError); ok {
					rErr = re
				} else {
					rErr = &RuntimeError{
						Pos:     Position{Line: 1, Col: 1},
						Message: err.Error(),
					}
				}

				if rErr.Pos.Line > 0 {
					errorLines = append(errorLines, rErr.Pos.Line)
				}

				runtimeDiag := Diagnostic{
					Pos:     rErr.Pos,
					Length:  1,
					Message: "runtime error: " + rErr.Message,
				}
				runtimeDiagStr := runtimeDiag.Format(p.srcLines)

				if diagStr != "" {
					diagStr += "\n" + runtimeDiagStr
				} else {
					diagStr = runtimeDiagStr
				}

				if stdout != "" {
					outStr = fmt.Sprintf("%s\nRuntimeError: Program crashed during execution:\n\n%s", stdout, runtimeDiagStr)
				} else {
					outStr = fmt.Sprintf("RuntimeError: Program crashed during execution:\n\n%s", runtimeDiagStr)
				}
			} else if val.Kind != ValNil && val.Kind != 0 {
				resStr = val.String()
			}
		}

		resp := APIResponse{
			Tokens:      tokStr,
			AST:         astStr,
			Diagnostics: diagStr,
			Output:      outStr,
			Result:      resStr,
			HasErrors:   hasErrors,
			ErrorLines:  errorLines,
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		zipPath := "robust-go-parser.zip"
		if _, err := os.Stat(zipPath); os.IsNotExist(err) {
			zipPath = "d:/Downloads/robust-go-parser.zip"
		}
		w.Header().Set("Content-Disposition", "attachment; filename=robust-go-parser.zip")
		w.Header().Set("Content-Type", "application/zip")
		http.ServeFile(w, r, zipPath)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, getPlaygroundHTML())
	})

	fmt.Printf("Dynamic Go Parser & Playground server running at http://localhost%s\n", addr)
	fmt.Printf("Press Ctrl+C to stop.\n")
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
	}
}

const defaultProgram = `package main

import "fmt"

func main() {
	var n int

	fmt.Print("Enter a number: ")
	fmt.Scan(&n)

	if n%2 == 0 {
		fmt.Println(n, "is even")
	} else {
		fmt.Println(n, "is odd")
	}

	sum := 0
	for i := 1; i <= n; i++ {
		sum += i
	}

	fmt.Println("Sum:", sum)
}`

const defaultStdin = "5"

var presetSamples = map[string]struct {
	Code  string `json:"code"`
	Stdin string `json:"stdin"`
}{
	"basic": {
		Code:  defaultProgram,
		Stdin: defaultStdin,
	},
	"realtime": {
		Code: `package main

type Counter struct {
	total int
}

func (c Counter) Add(val int) int {
	return c.total + val
}

func swapAndDivide(a, b int) (int, int) {
	return b, a / b
}

func main() {
	// 1. Multi-assign and simultaneous swapping
	x, y := 10, 20
	x, y = y, x
	fmt.Printf("1. Swapped: x=%d y=%d\n", x, y)

	// 2. Multi-return function
	s, d := swapAndDivide(5, 25)
	fmt.Printf("2. SwapAndDivide: s=%d d=%d\n", s, d)

	// 3. Real-time Map operations
	m := make(map[string]int)
	m["golang"] = 2009
	m["antigravity"] = 2026
	delete(m, "golang")
	fmt.Printf("3. Map len: %d, antigravity: %d\n", len(m), m["antigravity"])

	// 4. Struct receiver methods
	c := Counter{total: 100}
	fmt.Printf("4. Counter.Add(50): %d\n", c.Add(50))

	// 5. Standard library calls
	fmt.Printf("5. Math Sqrt(144): %.1f\n", math.Sqrt(144.0))
	fmt.Printf("6. Strings ToUpper: %s\n", strings.ToUpper("realtime go engine"))
	fmt.Printf("7. Strconv Itoa: %s\n", strconv.Itoa(9999))

	// 6. Go 1.22+ Integer range
	sum := 0
	for i := range 5 {
		sum += i
	}
	fmt.Printf("8. Range 5 sum: %d\n", sum)
}`,
		Stdin: "",
	},
	"structs": {
		Code: `package main

type User struct {
	Name string
	Age  int
	Role string
}

func main() {
	u := User{Name: "Alice", Age: 24, Role: "Engineer"}
	println("Initial:", u.Name, u.Age, u.Role)

	u.Age = 25
	u.Role = "Tech Lead"
	println("Updated:", u.Name, u.Age, u.Role)
}`,
		Stdin: "",
	},
	"switch": {
		Code: `package main

func main() {
	score := 85
	switch {
	case score >= 90:
		println("Grade: A")
	case score >= 80:
		println("Grade: B")
	case score >= 70:
		println("Grade: C")
	default:
		println("Grade: F")
	}

	day := 3
	switch day {
	case 1, 2, 3, 4, 5:
		println("Day", day, "is a Weekday")
	case 6, 7:
		println("Day", day, "is a Weekend")
	}
}`,
		Stdin: "",
	},
	"slices": {
		Code: `package main

func main() {
	nums := []int{10, 20, 30, 40, 50}
	println("Original slice length:", len(nums))

	sum := 0
	for idx, val := range nums {
		println("nums[", idx, "] =", val)
		sum += val
	}
	println("Total Sum:", sum)

	sub := nums[1:4]
	println("Subslice [1:4]:", sub[0], sub[1], sub[2])

	extended := append(sub, 999)
	println("Appended slice length:", len(extended))
	println("Appended item:", extended[3])
}`,
		Stdin: "",
	},
	"defer": {
		Code: `package main

func cleanup() {
	defer println("Cleanup: Step 1 (executed last)")
	defer println("Cleanup: Step 2")
	defer println("Cleanup: Step 3 (executed first)")
	println("--> Main operation working...")
}

func main() {
	cleanup()
	println("--> Function returned successfully")
}`,
		Stdin: "",
	},
	"recovery": {
		Code: `package main

var x int = 
var y int 10
func calc(a int, ) int {
	if a > 0 
		return a
	}
	return
}`,
		Stdin: "",
	},
}

func getPlaygroundHTML() string {
	codeBytes, _ := json.Marshal(defaultProgram)
	stdinBytes, _ := json.Marshal(defaultStdin)
	presetsBytes, _ := json.Marshal(presetSamples)
	grammarData, err := os.ReadFile("grammar.ebnf")
	var grammarStr string
	if err == nil {
		grammarStr = string(grammarData)
	} else {
		grammarStr = "(* EBNF Grammar available in GRAMMAR.md and grammar.ebnf *)"
	}
	grammarBytes, _ := json.Marshal(grammarStr)

	html := strings.Replace(htmlPlaygroundTemplate, "/*DEFAULT_CODE_PLACEHOLDER*/", string(codeBytes), 1)
	html = strings.Replace(html, "/*DEFAULT_STDIN_PLACEHOLDER*/", string(stdinBytes), 1)
	html = strings.Replace(html, "/*PRESET_SAMPLES_PLACEHOLDER*/", string(presetsBytes), 1)
	return strings.Replace(html, "/*GRAMMAR_TEXT_PLACEHOLDER*/", string(grammarBytes), 1)
}

const htmlPlaygroundTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Robust Go Parser & AST Engine</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, monospace; background: #0f172a; color: #f8fafc; display: flex; flex-direction: column; height: 100vh; }
  header { background: #1e293b; padding: 12px 20px; border-bottom: 1px solid #334155; display: flex; align-items: center; justify-content: space-between; }
  .header-left { display: flex; align-items: baseline; gap: 12px; }
  header h1 { font-size: 1.15rem; color: #38bdf8; font-weight: 600; letter-spacing: 0.5px; }
  .header-sub { color: #64748b; font-size: 0.82rem; }
  .btn-group { display: flex; gap: 10px; align-items: center; }
  button { background: #334155; color: #f8fafc; border: 1px solid #475569; padding: 7px 14px; border-radius: 6px; cursor: pointer; font-size: 0.85rem; font-family: inherit; font-weight: 500; transition: all 0.15s ease; }
  button:hover { background: #475569; }
  button.primary { background: #0284c7; border-color: #38bdf8; font-weight: 600; color: white; }
  button.primary:hover { background: #0369a1; }
  .container { display: flex; flex: 1; overflow: hidden; }
  .pane { flex: 1; display: flex; flex-direction: column; border-right: 1px solid #334155; }
  .pane:last-child { border-right: none; }
  .pane-header { background: #1e293b; padding: 9px 16px; font-size: 0.8rem; font-weight: 600; color: #94a3b8; letter-spacing: 0.5px; border-bottom: 1px solid #334155; display: flex; justify-content: space-between; align-items: center; }
  textarea { background: #090d16; color: #e2e8f0; font-family: "JetBrains Mono", Consolas, monospace; font-size: 0.95rem; padding: 16px; border: none; resize: none; outline: none; line-height: 1.5; tab-size: 4; }
  .editor-wrapper { position: relative; display: flex; flex: 3; background: #090d16; overflow: hidden; }
  .line-numbers {
    width: 48px;
    min-width: 48px;
    background: #0d1322;
    color: #475569;
    font-family: "JetBrains Mono", Consolas, monospace;
    font-size: 0.95rem;
    line-height: 1.5;
    padding: 16px 0 16px 0;
    text-align: right;
    user-select: none;
    border-right: 1px solid #1e293b;
    overflow: hidden;
    box-sizing: border-box;
  }
  .line-num {
    display: block;
    padding: 0 10px 0 6px;
    height: 1.5em;
    line-height: 1.5em;
    box-sizing: border-box;
    transition: all 0.15s ease;
  }
  .line-num.has-error {
    color: #f87171 !important;
    font-weight: 700;
    background: rgba(239, 68, 68, 0.22);
    border-left: 3px solid #ef4444;
    padding-left: 3px;
  }
  .code-area {
    flex: 1;
    background: transparent !important;
    color: #e2e8f0;
    font-family: "JetBrains Mono", Consolas, monospace;
    font-size: 0.95rem;
    line-height: 1.5;
    padding: 16px;
    border: none;
    resize: none;
    outline: none;
    tab-size: 4;
    overflow-x: auto;
    overflow-y: auto;
    white-space: pre;
    word-wrap: normal;
    box-sizing: border-box;
  }
  .stdin-area {
    background: #0c1222;
    color: #e2e8f0;
    font-family: "JetBrains Mono", Consolas, monospace;
    font-size: 0.95rem;
    line-height: 1.5;
    padding: 16px;
    border: none;
    resize: none;
    outline: none;
    tab-size: 4;
    min-height: 70px;
    max-height: 120px;
    flex: 1;
    border-top: 1px solid #1e293b;
    box-sizing: border-box;
  }
  .tabs { display: flex; gap: 4px; }
  .tab-btn { background: transparent; border: none; padding: 4px 10px; border-radius: 4px; color: #94a3b8; font-size: 0.8rem; cursor: pointer; }
  .tab-btn.active { background: #334155; color: #38bdf8; font-weight: 600; }
  pre { flex: 1; background: #090d16; color: #f1f5f9; padding: 16px; overflow: auto; font-family: "JetBrains Mono", Consolas, monospace; font-size: 0.92rem; line-height: 1.5; white-space: pre; }
  .diag { color: #f87171; }
  .ast { color: #67e8f9; }
  .tokens { color: #38bdf8; }
  .exec { color: #4ade80; }
  .shortcut { font-size: 0.75rem; color: #cbd5e1; margin-left: 4px; font-weight: normal; }
</style>
</head>
<body>
<header>
  <div class="header-left">
    <h1>Robust Go Parser &amp; AST Engine</h1>
    <span class="header-sub">Pratt Expression Parser &amp; Panic-Mode Recovery</span>
  </div>
  <div class="btn-group">
    <select id="sample-select" onchange="loadPreset(this.value)" style="background:#1e293b;color:#38bdf8;border:1px solid #0284c7;padding:6px 12px;border-radius:6px;font-family:inherit;font-size:0.85rem;cursor:pointer;outline:none;font-weight:600;">
      <option value="basic">Sample: Scan &amp; Math</option>
      <option value="realtime">Sample: Real-time Go (Maps, Methods, StdLib)</option>
      <option value="structs">Sample: Structs &amp; Fields (Phase 1)</option>
      <option value="switch">Sample: Switch Statement (Phase 1)</option>
      <option value="slices">Sample: For-Range &amp; Slices (Phase 1)</option>
      <option value="defer">Sample: Defer LIFO (Phase 1)</option>
      <option value="recovery">Sample: Slide 4 Broken Code</option>
    </select>
    <button onclick="clearAll()">Clear</button>
    <button onclick="resetDefault()">Reset</button>
    <a href="/download" download="robust-go-parser.zip" style="text-decoration:none;"><button type="button">⬇ Download ZIP</button></a>
    <button class="primary" onclick="runCode()">▶ Run / Parse <span class="shortcut">(Ctrl+Enter)</span></button>
  </div>
</header>
<div class="container">
  <div class="pane">
    <div class="pane-header">SOURCE CODE</div>
    <div class="editor-wrapper">
      <div class="line-numbers" id="line-numbers">1</div>
      <textarea id="code" class="code-area" spellcheck="false"></textarea>
    </div>
    <div class="pane-header" style="border-top: 1px solid #334155;">PROGRAM INPUT (STDIN)</div>
    <textarea id="stdin" class="stdin-area" spellcheck="false" placeholder="Enter input for fmt.Scan (e.g. 5)"></textarea>
  </div>
  <div class="pane">
    <div class="pane-header">
      <span>OUTPUT</span>
      <div class="tabs">
        <button class="tab-btn active" data-tab="exec" onclick="switchTab('exec')">Output</button>
        <button class="tab-btn" data-tab="ast" onclick="switchTab('ast')">AST Tree</button>
        <button class="tab-btn" data-tab="tokens" onclick="switchTab('tokens')">Tokens</button>
        <button class="tab-btn" data-tab="diag" id="tab-diag" onclick="switchTab('diag')">Diagnostics</button>
        <button class="tab-btn" data-tab="grammar" onclick="switchTab('grammar')">Grammar</button>
      </div>
    </div>
    <pre id="output" class="exec"></pre>
  </div>
</div>
<script>
const defaultCode = /*DEFAULT_CODE_PLACEHOLDER*/;
const defaultStdin = /*DEFAULT_STDIN_PLACEHOLDER*/;
const grammarText = /*GRAMMAR_TEXT_PLACEHOLDER*/;

let currentTab = 'exec';
let lastResponse = null;

let currentErrorLines = [];

function updateLineNumbers(errorLines = null) {
  if (errorLines !== null) {
    currentErrorLines = errorLines;
  }
  const code = document.getElementById('code').value;
  const lineCount = Math.max(1, code.split('\n').length);
  const lineNumbersEl = document.getElementById('line-numbers');
  const errSet = new Set(currentErrorLines);
  let html = '';
  for (let i = 1; i <= lineCount; i++) {
    const isErr = errSet.has(i);
    const cls = isErr ? 'line-num has-error' : 'line-num';
    const title = isErr ? ' title="Syntax or indentation error on line ' + i + '"' : '';
    html += '<span class="' + cls + '"' + title + '>' + i + '</span>';
  }
  lineNumbersEl.innerHTML = html;
  lineNumbersEl.scrollTop = document.getElementById('code').scrollTop;
}

function clearAll() {
  document.getElementById('code').value = '';
  document.getElementById('stdin').value = '';
  localStorage.removeItem('go_code');
  localStorage.removeItem('go_stdin');
  document.getElementById('output').textContent = '';
  lastResponse = null;
  const diagBtn = document.getElementById('tab-diag');
  diagBtn.textContent = 'Diagnostics';
  diagBtn.style.color = '';
  updateLineNumbers([]);
}

function switchTab(tab) {
  currentTab = tab;
  document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.tab === tab);
  });
  renderOutput();
}

async function runCode() {
  const code = document.getElementById('code').value;
  const stdin = document.getElementById('stdin').value;
  try {
    const res = await fetch('/api/run', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code, stdin })
    });
    lastResponse = await res.json();
    const diagBtn = document.getElementById('tab-diag');
    if (lastResponse.has_errors) {
      const diagCount = (lastResponse.diagnostics.match(/error\[/g) || []).length;
      diagBtn.textContent = 'Diagnostics (' + diagCount + ')';
      diagBtn.style.color = '#f87171';
      currentTab = 'exec';
      document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.tab === 'exec');
      });
      updateLineNumbers(lastResponse.error_lines || []);
    } else {
      diagBtn.textContent = 'Diagnostics';
      diagBtn.style.color = '';
      updateLineNumbers([]);
    }
    renderOutput();
  } catch (err) {
    document.getElementById('output').textContent = 'Error: ' + err;
  }
}

function renderOutput() {
  const el = document.getElementById('output');
  if (currentTab === 'grammar') {
    el.className = 'ast';
    el.textContent = grammarText;
    return;
  }
  if (!lastResponse) return;
  el.className = '';
  if (currentTab === 'exec') {
    if (lastResponse.has_errors) {
      el.className = 'diag';
      el.textContent = lastResponse.output;
    } else {
      el.className = 'exec';
      let text = '';
      if (lastResponse.output) text += lastResponse.output + '\n';
      if (lastResponse.result) text += '=> Result: ' + lastResponse.result;
      if (!text) text = '(Program completed with no output)';
      el.textContent = text;
    }
  } else if (currentTab === 'ast') {
    el.className = 'ast';
    el.textContent = lastResponse.ast || '(No AST generated)';
  } else if (currentTab === 'tokens') {
    el.className = 'tokens';
    el.textContent = lastResponse.tokens || '(No tokens)';
  } else if (currentTab === 'diag') {
    if (lastResponse.has_errors) {
      el.className = 'diag';
      el.textContent = lastResponse.diagnostics;
    } else {
      el.className = 'exec';
      el.textContent = '✓ No syntax errors found. Source parsed cleanly.';
    }
  }
}

function resetDefault() {
  document.getElementById('sample-select').value = 'basic';
  document.getElementById('code').value = defaultCode;
  document.getElementById('stdin').value = defaultStdin;
  localStorage.removeItem('go_code');
  localStorage.removeItem('go_stdin');
  updateLineNumbers([]);
  runCode();
}

const presetSamples = /*PRESET_SAMPLES_PLACEHOLDER*/;

function loadPreset(key) {
  const p = presetSamples[key];
  if (!p) return;
  document.getElementById('code').value = p.code;
  document.getElementById('stdin').value = p.stdin || '';
  localStorage.setItem('go_code', p.code);
  localStorage.setItem('go_stdin', p.stdin || '');
  updateLineNumbers([]);
  runCode();
}

window.onload = () => {
  const savedCode = localStorage.getItem('go_code');
  const savedStdin = localStorage.getItem('go_stdin');
  document.getElementById('code').value = savedCode !== null ? savedCode : defaultCode;
  document.getElementById('stdin').value = savedStdin !== null ? savedStdin : defaultStdin;
  updateLineNumbers();
  runCode();
};

const codeArea = document.getElementById('code');
const lineNumbersEl = document.getElementById('line-numbers');

codeArea.addEventListener('scroll', () => {
  lineNumbersEl.scrollTop = codeArea.scrollTop;
});

codeArea.addEventListener('input', (e) => {
  localStorage.setItem('go_code', e.target.value);
  updateLineNumbers();
});

codeArea.addEventListener('paste', () => {
  setTimeout(updateLineNumbers, 0);
});

document.getElementById('stdin').addEventListener('input', (e) => {
  localStorage.setItem('go_stdin', e.target.value);
});

codeArea.addEventListener('keydown', function(e) {
  if (e.key === 'Tab') {
    e.preventDefault();
    const start = this.selectionStart;
    const end = this.selectionEnd;
    this.value = this.value.substring(0, start) + "\t" + this.value.substring(end);
    this.selectionStart = this.selectionEnd = start + 1;
    localStorage.setItem('go_code', this.value);
    updateLineNumbers();
  }
});

document.addEventListener('keydown', (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
    e.preventDefault();
    runCode();
  }
});
</script>
</body>
</html>`

func reorderArgs(args []string) []string {
	if len(args) <= 1 {
		return args
	}
	var flags []string
	var nonFlags []string
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if (arg == "-serve" || arg == "--serve" || arg == "-code" || arg == "--code" || arg == "-input" || arg == "--input") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
		} else {
			nonFlags = append(nonFlags, arg)
		}
	}
	res := []string{args[0]}
	res = append(res, flags...)
	res = append(res, nonFlags...)
	return res
}

func main() {
	os.Args = reorderArgs(os.Args)
	tokensFlag := flag.Bool("tokens", false, "Dump the token stream table")
	astFlag := flag.Bool("ast", false, "Print the ASCII AST tree")
	diagFlag := flag.Bool("diag", false, "Print panic-mode diagnostics")
	execFlag := flag.Bool("exec", false, "Dynamically execute the Go source code")
	replFlag := flag.Bool("repl", false, "Start interactive dynamic REPL session")
	serveFlag := flag.String("serve", "", "Start dynamic web playground server (e.g. :8080)")
	demoFlag := flag.Bool("demo", false, "Run the live demo showcasing all features")
	codeFlag := flag.String("code", "", "Inline Go code snippet to parse")
	inputFlag := flag.String("input", "", "Dynamic standard input (stdin) for execution")

	flag.Parse()

	if *demoFlag {
		runDemo()
		return
	}

	if *replFlag {
		runREPL()
		return
	}

	if *serveFlag != "" {
		addr := *serveFlag
		if !strings.HasPrefix(addr, ":") {
			addr = ":" + addr
		}
		runServer(addr)
		return
	}

	var src string
	if *codeFlag != "" {
		src = *codeFlag
	} else if flag.NArg() > 0 {
		filePath := flag.Arg(0)
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filePath, err)
			os.Exit(1)
		}
		src = string(data)
	} else {
		// Read from stdin if available
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			bytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
				os.Exit(1)
			}
			src = string(bytes)
		} else {
			// No input provided, start REPL
			fmt.Println("No input file provided. Starting dynamic REPL (use -help for options)...")
			runREPL()
			return
		}
	}

	tokens := Tokenize(src)
	if *tokensFlag {
		fmt.Println("[Token Stream]:")
		printTokenStream(tokens)
		fmt.Println()
	}

	parser := NewParser(src)
	file := parser.ParseFile()

	hasErrors := len(parser.errors) > 0

	if hasErrors {
		fmt.Fprintf(os.Stderr, "SyntaxError: Failed to parse source code (%d error(s) found):\n\n", len(parser.errors))
		fmt.Fprint(os.Stderr, parser.FormatDiagnostics())
		if !*diagFlag && !*astFlag && !*tokensFlag {
			os.Exit(1)
		}
	} else if *diagFlag {
		fmt.Println("No errors found.")
	}

	if *astFlag {
		fmt.Println("[AST Tree]:")
		if len(file.Decls) == 0 && len(file.Stmts) == 1 && file.Name == nil {
			if exprStmt, ok := file.Stmts[0].(*ExprStmt); ok {
				fmt.Println(PrintAST(exprStmt.X))
			} else {
				fmt.Println(PrintAST(file.Stmts[0]))
			}
		} else {
			fmt.Println(PrintAST(file))
		}
	}

	if !hasErrors && (*execFlag || (!*astFlag && !*tokensFlag && !*diagFlag)) {
		var stdinReader io.Reader = os.Stdin
		var directStdout io.Writer = os.Stdout
		if *inputFlag != "" {
			stdinReader = strings.NewReader(*inputFlag)
			directStdout = nil
		}
		interp := NewInterpreterWithIO(stdinReader, directStdout)
		val, stdout, err := interp.ExecuteFile(file)
		if err != nil {
			if rErr, ok := err.(*RuntimeError); ok {
				diag := Diagnostic{
					Pos:     rErr.Pos,
					Length:  1,
					Message: "runtime error: " + rErr.Message,
				}
				fmt.Fprintln(os.Stderr, diag.Format(parser.srcLines))
			} else {
				fmt.Fprintf(os.Stderr, "Runtime Error: %v\n", err)
			}
			os.Exit(1)
		}
		if interp.GlobalEnv.directStdout == nil && stdout != "" {
			fmt.Print(stdout)
		}
		if val.Kind != ValNil && val.Kind != 0 {
			fmt.Printf("=> Result: %s\n", val)
		}
	}
}
