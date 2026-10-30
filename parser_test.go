package main

import (
	"strings"
	"testing"
)

func TestParseValidGoProgram(t *testing.T) {
	src := `package main

var limit int = 10

func calc(a, b int) int {
	x := a + b*2
	x = x - (limit / 2)
	if x > 5 {
		return x
	}
	return -x
}
`
	parser := NewParser(src)
	file := parser.ParseFile()

	if len(parser.errors) > 0 {
		t.Fatalf("unexpected parser errors:\n%s", parser.FormatDiagnostics())
	}

	if file.Name == nil || file.Name.Name != "main" {
		t.Fatalf("expected package main, got %v", file.Name)
	}

	if len(file.Decls) != 2 {
		t.Fatalf("expected 2 declarations, got %d", len(file.Decls))
	}

	// 1. var limit int = 10
	varDecl, ok := file.Decls[0].(*VarDecl)
	if !ok || varDecl.Name.Name != "limit" {
		t.Fatalf("expected VarDecl limit, got %T", file.Decls[0])
	}

	// 2. func calc(a, b int) int
	funcDecl, ok := file.Decls[1].(*FuncDecl)
	if !ok || funcDecl.Name.Name != "calc" {
		t.Fatalf("expected FuncDecl calc, got %T", file.Decls[1])
	}
	if len(funcDecl.Params) != 1 || len(funcDecl.Params[0].Names) != 2 {
		t.Fatalf("expected 2 param names (a, b), got %v", funcDecl.Params)
	}
	if funcDecl.Body == nil || len(funcDecl.Body.List) != 4 {
		t.Fatalf("expected 4 statements in func calc, got %d", len(funcDecl.Body.List))
	}

	// Check AST tree contains expected structures
	tree := PrintAST(file)
	if !strings.Contains(tree, "VarDecl limit") {
		t.Errorf("expected VarDecl limit in tree output")
	}
	if !strings.Contains(tree, "FuncDecl calc") {
		t.Errorf("expected FuncDecl calc in tree output")
	}
	if !strings.Contains(tree, "IfStmt") {
		t.Errorf("expected IfStmt in tree output")
	}
}

func TestASTTreeFormattingSlide5b(t *testing.T) {
	t.Run("AssignStmt tree matches Slide 5b", func(t *testing.T) {
		src := "x := a + b*2\n"
		p := NewParser(src)
		file := p.ParseFile()
		if len(file.Stmts) == 0 {
			t.Fatalf("expected at least 1 statement")
		}
		tree := PrintAST(file.Stmts[0])
		expectedSubstrings := []string{
			"AssignStmt (1:1)",
			"├── Lhs: Ident x",
			"└── Rhs: Binary +",
			"    ├── X: Ident a",
			"    └── Y: Binary *",
			"        ├── X: Ident b",
			"        └── Y: Int 2",
		}
		for _, substr := range expectedSubstrings {
			if !strings.Contains(tree, substr) {
				t.Errorf("expected tree to contain %q, but got:\n%s", substr, tree)
			}
		}
	})

	t.Run("Binary paren tree matches Slide 5b", func(t *testing.T) {
		src := "(a + b) * c\n"
		p := NewParser(src)
		expr := p.parseExpr(0)
		tree := PrintAST(expr)
		expectedSubstrings := []string{
			"Binary *",
			"├── X: Paren",
			"│   └── Binary +",
			"│       ├── X: Ident a",
			"│       └── Y: Ident b",
			"└── Y: Ident c",
		}
		for _, substr := range expectedSubstrings {
			if !strings.Contains(tree, substr) {
				t.Errorf("expected tree to contain %q, but got:\n%s", substr, tree)
			}
		}
	})
}

func TestForLoopStatements(t *testing.T) {
	t.Run("for condition", func(t *testing.T) {
		src := "for x < 10 { x = x + 1 }"
		p := NewParser(src)
		file := p.ParseFile()
		if len(p.errors) > 0 {
			t.Fatalf("unexpected errors: %v", p.errors)
		}
		if len(file.Stmts) != 1 {
			t.Fatalf("expected 1 stmt, got %d", len(file.Stmts))
		}
		forStmt, ok := file.Stmts[0].(*ForStmt)
		if !ok || forStmt.Cond == nil {
			t.Fatalf("expected ForStmt with condition")
		}
	})

	t.Run("infinite for loop", func(t *testing.T) {
		src := "for { return }"
		p := NewParser(src)
		file := p.ParseFile()
		if len(p.errors) > 0 {
			t.Fatalf("unexpected errors: %v", p.errors)
		}
		forStmt, ok := file.Stmts[0].(*ForStmt)
		if !ok || forStmt.Cond != nil {
			t.Fatalf("expected ForStmt without condition")
		}
	})
}

func TestParseUserGoProgram(t *testing.T) {
	src := `package main

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
}
`
	p := NewParser(src)
	file := p.ParseFile()

	if len(p.errors) > 0 {
		t.Fatalf("expected 0 errors on valid user program, got %d:\n%s", len(p.errors), p.FormatDiagnostics())
	}

	if len(file.Decls) != 2 {
		t.Fatalf("expected 2 decls (import, func main), got %d", len(file.Decls))
	}

	importDecl, ok := file.Decls[0].(*ImportDecl)
	if !ok || len(importDecl.Specs) != 1 || importDecl.Specs[0].Path.Value != `"fmt"` {
		t.Fatalf("expected ImportDecl with \"fmt\", got %v", file.Decls[0])
	}

	fnDecl, ok := file.Decls[1].(*FuncDecl)
	if !ok || fnDecl.Name.Name != "main" {
		t.Fatalf("expected FuncDecl main, got %v", file.Decls[1])
	}

	tree := PrintAST(file)
	if !strings.Contains(tree, `ImportSpec "fmt"`) {
		t.Errorf("expected tree to contain ImportSpec, got:\n%s", tree)
	}
	if !strings.Contains(tree, `IncDecStmt ++`) {
		t.Errorf("expected tree to contain IncDecStmt ++, got:\n%s", tree)
	}
}
