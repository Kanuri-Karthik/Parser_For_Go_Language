package main

import (
	"strings"
	"testing"
)

func TestPhase1StructDeclarationAndCompositeLit(t *testing.T) {
	src := `package main

type Point struct {
	X int
	Y int
}

func main() {
	p := Point{X: 10, Y: 20}
	p.X = 55
	println(p.X, p.Y)
}
`
	p := NewParser(src)
	file := p.ParseFile()
	if len(p.errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.FormatDiagnostics())
	}

	interp := NewInterpreter()
	_, out, err := interp.ExecuteFile(file)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	expected := "55 20\n"
	if out != expected {
		t.Fatalf("expected output %q, got %q", expected, out)
	}
}

func TestPhase1SwitchStatement(t *testing.T) {
	src := `package main

func main() {
	x := 2
	switch x {
	case 1:
		println("one")
	case 2, 3:
		println("two or three")
	default:
		println("other")
	}

	y := 15
	switch {
	case y < 10:
		println("small")
	case y >= 10:
		println("big")
	}
}
`
	p := NewParser(src)
	file := p.ParseFile()
	if len(p.errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.FormatDiagnostics())
	}

	interp := NewInterpreter()
	_, out, err := interp.ExecuteFile(file)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if !strings.Contains(out, "two or three") {
		t.Fatalf("expected 'two or three' in output, got %q", out)
	}
	if !strings.Contains(out, "big") {
		t.Fatalf("expected 'big' in output, got %q", out)
	}
}

func TestPhase1ForRangeLoop(t *testing.T) {
	src := `package main

func main() {
	items := []int{10, 20, 30}
	sum := 0
	for i, v := range items {
		sum += v
	}
	println("Sum:", sum)
}
`
	p := NewParser(src)
	file := p.ParseFile()
	if len(p.errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.FormatDiagnostics())
	}

	interp := NewInterpreter()
	_, out, err := interp.ExecuteFile(file)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	expected := "Sum: 60\n"
	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestPhase1SlicingAndAppend(t *testing.T) {
	src := `package main

func main() {
	numbers := []int{100, 200, 300, 400, 500}
	sub := numbers[1:4]
	println("Sub length:", len(sub))
	sub[0] = 999
	println("Modified sub:", sub[0], sub[1], sub[2])

	grown := append(sub, 777)
	println("Grown length:", len(grown))
	println("Last element:", grown[3])
}
`
	p := NewParser(src)
	file := p.ParseFile()
	if len(p.errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.FormatDiagnostics())
	}

	interp := NewInterpreter()
	_, out, err := interp.ExecuteFile(file)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if !strings.Contains(out, "Sub length: 3") {
		t.Fatalf("expected sub length 3, got: %q", out)
	}
	if !strings.Contains(out, "Modified sub: 999 300 400") {
		t.Fatalf("expected modified sub elements, got: %q", out)
	}
	if !strings.Contains(out, "Grown length: 4") {
		t.Fatalf("expected grown length 4, got: %q", out)
	}
	if !strings.Contains(out, "Last element: 777") {
		t.Fatalf("expected last element 777, got: %q", out)
	}
}

func TestPhase1DeferExecutionLIFO(t *testing.T) {
	src := `package main

func testDefer() {
	defer println("deferred 1")
	defer println("deferred 2")
	println("body executed")
}

func main() {
	testDefer()
}
`
	p := NewParser(src)
	file := p.ParseFile()
	if len(p.errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.FormatDiagnostics())
	}

	interp := NewInterpreter()
	_, out, err := interp.ExecuteFile(file)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	expected := "body executed\ndeferred 2\ndeferred 1\n"
	if out != expected {
		t.Fatalf("expected LIFO order %q, got %q", expected, out)
	}
}

func TestPhase1RawStringsAndRunes(t *testing.T) {
	src := "package main\nfunc main() {\n\ts := `raw line 1\\nline 2`\n\tr := 'A'\n\tprintln(s, r)\n}\n"
	p := NewParser(src)
	file := p.ParseFile()
	if len(p.errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", p.FormatDiagnostics())
	}

	interp := NewInterpreter()
	_, out, err := interp.ExecuteFile(file)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if !strings.Contains(out, "65") {
		t.Fatalf("expected rune 65 in output, got: %q", out)
	}
	if !strings.Contains(out, "raw line 1\\nline 2") {
		t.Fatalf("expected raw string in output, got: %q", out)
	}
}
