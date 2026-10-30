//go:build js

package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"
)

//go:embed grammar.ebnf
var embeddedGrammar string

// wasmResponse mirrors APIResponse for JSON marshalling.
type wasmResponse struct {
	Tokens      string `json:"tokens"`
	AST         string `json:"ast"`
	Diagnostics string `json:"diagnostics"`
	Output      string `json:"output"`
	Result      string `json:"result"`
	HasErrors   bool   `json:"has_errors"`
	ErrorLines  []int  `json:"error_lines"`
}

// formatTokenStreamWasm formats the token table to a string.
func formatTokenStreamWasm(tokens []Token) string {
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

// jsParseAndRun is the JS-callable function: goParseAndRun(code, stdin) → JSON string.
func jsParseAndRun(this js.Value, args []js.Value) interface{} {
	code := ""
	stdin := ""
	if len(args) > 0 {
		code = args[0].String()
	}
	if len(args) > 1 {
		stdin = args[1].String()
	}

	// Tokenise
	tokens := Tokenize(code)
	tokStr := formatTokenStreamWasm(tokens)

	// Parse
	p := NewParser(code)
	file := p.ParseFile()

	diagStr := p.FormatDiagnostics()
	hasErrors := len(p.errors) > 0

	// Build AST string
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

	var outStr, resStr string
	var errorLines []int

	for _, err := range p.errors {
		errorLines = append(errorLines, err.Pos.Line)
	}

	if hasErrors {
		outStr = fmt.Sprintf("SyntaxError: Failed to parse Go source (%d syntax error(s) found):\n\n%s", len(p.errors), diagStr)
	} else {
		interp := NewInterpreterWithIO(strings.NewReader(stdin), nil)
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
			if stdout != "" {
				outStr = fmt.Sprintf("%s\nRuntimeError: Program crashed during execution:\n\n%s", stdout, runtimeDiagStr)
			} else {
				outStr = fmt.Sprintf("RuntimeError: Program crashed during execution:\n\n%s", runtimeDiagStr)
			}
		} else if val.Kind != ValNil && val.Kind != 0 {
			resStr = val.String()
		}
	}

	resp := wasmResponse{
		Tokens:      tokStr,
		AST:         astStr,
		Diagnostics: diagStr,
		Output:      outStr,
		Result:      resStr,
		HasErrors:   hasErrors,
		ErrorLines:  errorLines,
	}
	b, _ := json.Marshal(resp)
	return js.ValueOf(string(b))
}

// jsGetGrammar exposes the embedded grammar text to JS.
func jsGetGrammar(this js.Value, args []js.Value) interface{} {
	return js.ValueOf(embeddedGrammar)
}

func main() {
	js.Global().Set("goParseAndRun", js.FuncOf(jsParseAndRun))
	js.Global().Set("goGetGrammar", js.FuncOf(jsGetGrammar))
	// Keep the Go runtime alive indefinitely.
	select {}
}
