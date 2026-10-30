package main

import (
	"strings"
	"testing"
)

func TestMissingClosingBrace(t *testing.T) {
	src := `package main

func main() {
	x := 10
`
	p := NewParser(src)
	_ = p.ParseFile()

	if len(p.errors) == 0 {
		t.Fatalf("expected error for unclosed brace, got 0")
	}

	foundBraceErr := false
	for _, err := range p.errors {
		if strings.Contains(err.Message, "expected '}'") {
			foundBraceErr = true
			break
		}
	}
	if !foundBraceErr {
		t.Fatalf("expected missing '}' diagnostic, got: %v", p.FormatDiagnostics())
	}
}

func TestAdjacentStatementsWithoutSemicolon(t *testing.T) {
	src := `package main

func main() {
	x := 10 y := 20
}
`
	p := NewParser(src)
	_ = p.ParseFile()

	if len(p.errors) == 0 {
		t.Fatalf("expected error for adjacent statements on same line, got 0")
	}

	foundSemiErr := false
	for _, err := range p.errors {
		if strings.Contains(err.Message, "expected ';'") {
			foundSemiErr = true
			if err.Pos.Line != 4 {
				t.Errorf("expected error on line 4, got line %d", err.Pos.Line)
			}
			break
		}
	}
	if !foundSemiErr {
		t.Fatalf("expected missing ';' diagnostic, got: %v", p.FormatDiagnostics())
	}
}

func TestVarDeclMissingTypeOrAssign(t *testing.T) {
	src := `package main

func main() {
	var a
}
`
	p := NewParser(src)
	_ = p.ParseFile()

	if len(p.errors) == 0 {
		t.Fatalf("expected error for 'var a' missing type/assign, got 0")
	}

	diag := p.FormatDiagnostics()
	if !strings.Contains(diag, "type or '='") {
		t.Fatalf("expected 'type or =' error, got: %s", diag)
	}
}

func TestConstDeclMissingAssign(t *testing.T) {
	src := `package main

func main() {
	const a int
}
`
	p := NewParser(src)
	_ = p.ParseFile()

	if len(p.errors) == 0 {
		t.Fatalf("expected error for 'const a int' missing assign, got 0")
	}

	diag := p.FormatDiagnostics()
	if !strings.Contains(diag, "expected '='") {
		t.Fatalf("expected 'expected =' error, got: %s", diag)
	}
}

func TestIllegalCharacterError(t *testing.T) {
	src := `package main

func main() {
	@
}
`
	p := NewParser(src)
	_ = p.ParseFile()

	if len(p.errors) == 0 {
		t.Fatalf("expected error for illegal character '@', got 0")
	}

	diag := p.FormatDiagnostics()
	if !strings.Contains(diag, "illegal character '@'") {
		t.Fatalf("expected 'illegal character @' in diag, got: %s", diag)
	}
}

func TestRuntimeErrorDivisionByZeroPosition(t *testing.T) {
	src := `package main

func main() {
	x := 10
	y := 0
	z := x / y
	println(z)
}
`
	p := NewParser(src)
	file := p.ParseFile()
	if len(p.errors) > 0 {
		t.Fatalf("unexpected parse error: %s", p.FormatDiagnostics())
	}

	interp := NewInterpreter()
	_, _, err := interp.ExecuteFile(file)
	if err == nil {
		t.Fatalf("expected runtime division by zero error, got nil")
	}

	rErr, ok := err.(*RuntimeError)
	if !ok {
		t.Fatalf("expected *RuntimeError, got %T (%v)", err, err)
	}

	if rErr.Pos.Line != 6 {
		t.Errorf("expected error on line 6, got line %d", rErr.Pos.Line)
	}
	if !strings.Contains(rErr.Message, "division by zero") {
		t.Errorf("expected 'division by zero' in error message, got %q", rErr.Message)
	}

	diag := Diagnostic{
		Pos:     rErr.Pos,
		Length:  1,
		Message: "runtime error: " + rErr.Message,
	}
	formatted := diag.Format(p.srcLines)
	if !strings.Contains(formatted, "error[6:") {
		t.Errorf("expected 'error[6:' in formatted diagnostic, got: %s", formatted)
	}
	if !strings.Contains(formatted, "z := x / y") {
		t.Errorf("expected source snippet in formatted diagnostic, got: %s", formatted)
	}
}

func TestRuntimeErrorIndexOutOfBoundsPosition(t *testing.T) {
	src := `package main

func main() {
	s := []int{10, 20}
	println(s[5])
}
`
	p := NewParser(src)
	file := p.ParseFile()
	if len(p.errors) > 0 {
		t.Fatalf("unexpected parse error: %s", p.FormatDiagnostics())
	}

	interp := NewInterpreter()
	_, _, err := interp.ExecuteFile(file)
	if err == nil {
		t.Fatalf("expected index out of range error, got nil")
	}

	rErr, ok := err.(*RuntimeError)
	if !ok {
		t.Fatalf("expected *RuntimeError, got %T (%v)", err, err)
	}

	if rErr.Pos.Line != 5 {
		t.Errorf("expected error on line 5, got line %d", rErr.Pos.Line)
	}
	if !strings.Contains(rErr.Message, "index out of range [5] with length 2") {
		t.Errorf("expected index out of range [5] with length 2, got %q", rErr.Message)
	}
}

func TestRuntimeErrorUndefinedIdentifierPosition(t *testing.T) {
	src := `package main

func main() {
	println(missingVar)
}
`
	p := NewParser(src)
	file := p.ParseFile()
	if len(p.errors) > 0 {
		t.Fatalf("unexpected parse error: %s", p.FormatDiagnostics())
	}

	interp := NewInterpreter()
	_, _, err := interp.ExecuteFile(file)
	if err == nil {
		t.Fatalf("expected undefined identifier error, got nil")
	}

	rErr, ok := err.(*RuntimeError)
	if !ok {
		t.Fatalf("expected *RuntimeError, got %T (%v)", err, err)
	}

	if rErr.Pos.Line != 4 {
		t.Errorf("expected error on line 4, got line %d", rErr.Pos.Line)
	}
	if !strings.Contains(rErr.Message, "undefined identifier: missingVar") {
		t.Errorf("expected undefined identifier missingVar, got %q", rErr.Message)
	}
}
