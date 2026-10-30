package main

import (
	"strings"
	"testing"
)

func TestPanicModeRecoverySlide4And5(t *testing.T) {
	invalidSrc := "x := ;\nvar y int 10\nz := (1 + ;\nfunc f( {\nif x > 5 return\n"

	parser := NewParser(invalidSrc)
	_ = parser.ParseFile()

	if len(parser.errors) != 5 {
		t.Fatalf("expected exactly 5 errors, got %d:\n%s", len(parser.errors), parser.FormatDiagnostics())
	}

	expectedErrors := []struct {
		line    int
		col     int
		message string
		found   string
	}{
		{1, 6, "expected expression", "';'"},
		{2, 11, "expected ';'", "INT(10)"},
		{3, 11, "expected expression", "';'"},
		{4, 9, "expected ')'", "'{'"},
		{5, 10, "expected '{'", "RETURN"},
	}

	for i, exp := range expectedErrors {
		err := parser.errors[i]
		if err.Pos.Line != exp.line || err.Pos.Col != exp.col {
			t.Errorf("[%d] expected pos %d:%d, got %d:%d", i, exp.line, exp.col, err.Pos.Line, err.Pos.Col)
		}
		if err.Message != exp.message {
			t.Errorf("[%d] expected message %q, got %q", i, exp.message, err.Message)
		}
		if err.Found != exp.found {
			t.Errorf("[%d] expected found %q, got %q", i, exp.found, err.Found)
		}
	}

	formatted := parser.FormatDiagnostics()
	if !strings.Contains(formatted, "5 errors, resynchronized each time") {
		t.Errorf("expected summary '5 errors, resynchronized each time' in diagnostics, got:\n%s", formatted)
	}
}
