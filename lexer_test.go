package main

import (
	"testing"
)

func TestLexerSingleExpression(t *testing.T) {
	input := "x := a + b*2\n"
	tokens := Tokenize(input)

	expected := []struct {
		tokType TokenType
		lexeme  string
		line    int
		col     int
	}{
		{IDENT, "x", 1, 1},
		{DEFINE, ":=", 1, 3},
		{IDENT, "a", 1, 6},
		{ADD, "+", 1, 8},
		{IDENT, "b", 1, 10},
		{MUL, "*", 1, 11},
		{INT, "2", 1, 12},
		{SEMICOLON, `\n auto`, 1, 13},
		{EOF, "EOF", 2, 1},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		got := tokens[i]
		if got.Type != exp.tokType {
			t.Errorf("[%d] expected token type %s, got %s", i, exp.tokType, got.Type)
		}
		if !got.IsAutoSemi && got.Type != EOF && got.Lexeme != exp.lexeme {
			t.Errorf("[%d] expected lexeme %q, got %q", i, exp.lexeme, got.Lexeme)
		}
		if got.Pos.Line != exp.line || got.Pos.Col != exp.col {
			t.Errorf("[%d] %s expected pos %d:%d, got %d:%d", i, got.Type, exp.line, exp.col, got.Pos.Line, got.Pos.Col)
		}
	}
}

func TestLexerAutomaticSemicolonRules(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		expectSemi bool
	}{
		{"after identifier", "foo\n", true},
		{"after integer", "42\n", true},
		{"after float", "3.14\n", true},
		{"after string", `"hello"` + "\n", true},
		{"after return", "return\n", true},
		{"after rparen", "(foo)\n", true},
		{"after rbrack", "arr[0]\n", true},
		{"after rbrace", "{}\n", true},
		{"after inc", "i++\n", true},
		{"after dec", "i--\n", true},
		{"not after plus", "1 +\n", false},
		{"not after func", "func\n", false},
		{"not after var", "var\n", false},
		{"not after lparen", "(\n", false},
		{"not after lbrace", "{\n", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens := Tokenize(tc.src)
			foundSemi := false
			for _, tok := range tokens {
				if tok.Type == SEMICOLON && tok.IsAutoSemi {
					foundSemi = true
					break
				}
			}
			if foundSemi != tc.expectSemi {
				t.Errorf("for %q, expected auto-semi %v, got %v", tc.src, tc.expectSemi, foundSemi)
			}
		})
	}
}

func TestLexerEdgeCases(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		tokens := Tokenize("")
		if len(tokens) != 1 || tokens[0].Type != EOF {
			t.Fatalf("expected 1 EOF token, got %v", tokens)
		}
	})

	t.Run("line comment", func(t *testing.T) {
		tokens := Tokenize("// line comment\nx := 1")
		if len(tokens) == 0 || tokens[0].Type != IDENT {
			t.Fatalf("expected IDENT x, got %v", tokens[0])
		}
	})

	t.Run("block comment with newline inserts semi", func(t *testing.T) {
		tokens := Tokenize("x /* comment\n newline */")
		foundSemi := false
		for _, tok := range tokens {
			if tok.Type == SEMICOLON {
				foundSemi = true
			}
		}
		if !foundSemi {
			t.Errorf("expected semicolon insertion after block comment with newline")
		}
	})
}
