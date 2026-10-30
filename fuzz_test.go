package main

import (
	"testing"
)

func FuzzParser(f *testing.F) {
	seeds := []string{
		"x := a + b*2",
		"(a + b) * c",
		"package main\nvar limit int = 10\nfunc calc(a, b int) int { return a + b }\n",
		"x := ;\nvar y int 10\nz := (1 + ;\nfunc f( {\nif x > 5 return\n",
		"for x > 0 { x = x - 1 }",
		"return -x",
		"",
		";;;",
		"func f() { if a { if b { return } } }",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		p := NewParser(input)
		file := p.ParseFile()
		if file != nil {
			_ = PrintAST(file)
		}
		_ = p.FormatDiagnostics()
	})
}
