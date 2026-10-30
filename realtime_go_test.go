package main

import (
	"testing"
)

func runGoSource(t *testing.T, src string) (Value, string) {
	t.Helper()
	parser := NewParser(src)
	file := parser.ParseFile()
	if len(parser.errors) > 0 {
		t.Fatalf("parse errors encountered:\n%s", parser.FormatDiagnostics())
	}
	interp := NewInterpreter()
	val, stdout, err := interp.ExecuteFile(file)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return val, stdout
}

// 1. Multi-variable short definition & simultaneous swapping
func TestRealtimeMultiAssignAndSwap(t *testing.T) {
	src := `package main
func main() {
	a, b := 10, 20
	a, b = b, a
	fmt.Printf("%d %d", a, b)
}
`
	_, out := runGoSource(t, src)
	if out != "20 10" {
		t.Fatalf("expected '20 10', got '%s'", out)
	}
}

// 2. Blank identifier discard
func TestRealtimeBlankIdentifier(t *testing.T) {
	src := `package main
func main() {
	x, _ := 100, 200
	_ = x
	fmt.Printf("%d", x)
}
`
	_, out := runGoSource(t, src)
	if out != "100" {
		t.Fatalf("expected '100', got '%s'", out)
	}
}

// 3. Multi-return function
func TestRealtimeMultiReturn(t *testing.T) {
	src := `package main
func divide(a, b int) (int, int) {
	return a / b, a % b
}
func main() {
	q, r := divide(17, 5)
	fmt.Printf("q=%d r=%d", q, r)
}
`
	_, out := runGoSource(t, src)
	if out != "q=3 r=2" {
		t.Fatalf("expected 'q=3 r=2', got '%s'", out)
	}
}

// 4. Map operations: make, write, read, delete, len, range
func TestRealtimeMapOperations(t *testing.T) {
	src := `package main
func main() {
	m := make(map[string]int)
	m["apple"] = 5
	m["banana"] = 8
	m["cherry"] = 12
	l1 := len(m)
	delete(m, "banana")
	l2 := len(m)
	fmt.Printf("l1=%d l2=%d val=%d", l1, l2, m["apple"])
}
`
	_, out := runGoSource(t, src)
	if out != "l1=3 l2=2 val=5" {
		t.Fatalf("expected 'l1=3 l2=2 val=5', got '%s'", out)
	}
}

// 5. Map range iteration
func TestRealtimeMapRange(t *testing.T) {
	src := `package main
func main() {
	m := map[string]int{"a": 1, "b": 2}
	out := ""
	for k, v := range m {
		out = out + k + ":" + strconv.Itoa(v) + " "
	}
	fmt.Print(strings.TrimSpace(out))
}
`
	_, out := runGoSource(t, src)
	if out != "a:1 b:2" {
		t.Fatalf("expected 'a:1 b:2', got '%s'", out)
	}
}

// 6. Const grouping with iota
func TestRealtimeConstIota(t *testing.T) {
	src := `package main
const (
	A = iota
	B
	C
)
func main() {
	fmt.Printf("%d %d %d", A, B, C)
}
`
	_, out := runGoSource(t, src)
	if out != "0 1 2" {
		t.Fatalf("expected '0 1 2', got '%s'", out)
	}
}

// 7. Grouped var declaration
func TestRealtimeGroupedVar(t *testing.T) {
	src := `package main
var (
	x = 10
	y = 20
)
func main() {
	fmt.Printf("%d", x + y)
}
`
	_, out := runGoSource(t, src)
	if out != "30" {
		t.Fatalf("expected '30', got '%s'", out)
	}
}

// 8. Struct methods
func TestRealtimeStructMethods(t *testing.T) {
	src := `package main
type Calculator struct {
	base int
}
func (c Calculator) Add(x int) int {
	return c.base + x
}
func main() {
	calc := Calculator{base: 50}
	ans := calc.Add(25)
	fmt.Printf("ans=%d", ans)
}
`
	_, out := runGoSource(t, src)
	if out != "ans=75" {
		t.Fatalf("expected 'ans=75', got '%s'", out)
	}
}

// 9. Numeric literal formats: hex, binary, octal, underscores, scientific
func TestRealtimeNumericLiterals(t *testing.T) {
	src := `package main
func main() {
	h := 0xFF
	b := 0b1011
	o := 0o77
	u := 1_000_000
	s := 1e3
	fmt.Printf("%d %d %d %d %g", h, b, o, u, s)
}
`
	_, out := runGoSource(t, src)
	if out != "255 11 63 1000000 1000" {
		t.Fatalf("expected '255 11 63 1000000 1000', got '%s'", out)
	}
}

// 10. Bitwise operations: &, |, ^, <<, >>, &^, unary ^
func TestRealtimeBitwiseOperations(t *testing.T) {
	src := `package main
func main() {
	a := 12  // 1100
	b := 10  // 1010
	and := a & b    // 8 (1000)
	or  := a | b    // 14 (1110)
	xor := a ^ b    // 6 (0110)
	shl := a << 1   // 24
	shr := a >> 1   // 6
	andNot := a &^ b // 4 (0100)
	inv := ^0        // -1
	fmt.Printf("%d %d %d %d %d %d %d", and, or, xor, shl, shr, andNot, inv)
}
`
	_, out := runGoSource(t, src)
	if out != "8 14 6 24 6 4 -1" {
		t.Fatalf("expected '8 14 6 24 6 4 -1', got '%s'", out)
	}
}

// 11. Standard library math package
func TestRealtimeMathPackage(t *testing.T) {
	src := `package main
func main() {
	sq := math.Sqrt(64.0)
	ab := math.Abs(-42.0)
	pw := math.Pow(2.0, 4.0)
	fmt.Printf("%.0f %.0f %.0f", sq, ab, pw)
}
`
	_, out := runGoSource(t, src)
	if out != "8 42 16" {
		t.Fatalf("expected '8 42 16', got '%s'", out)
	}
}

// 12. Standard library strings package
func TestRealtimeStringsPackage(t *testing.T) {
	src := `package main
func main() {
	has := strings.Contains("gopher", "ph")
	upper := strings.ToUpper("hello")
	lower := strings.ToLower("WORLD")
	trim := strings.TrimSpace("  spaced  ")
	fmt.Printf("%t %s %s %s", has, upper, lower, trim)
}
`
	_, out := runGoSource(t, src)
	if out != "true HELLO world spaced" {
		t.Fatalf("expected 'true HELLO world spaced', got '%s'", out)
	}
}

// 13. Standard library strconv package
func TestRealtimeStrconvPackage(t *testing.T) {
	src := `package main
func main() {
	s := strconv.Itoa(9876)
	n, _ := strconv.Atoi("1234")
	fmt.Printf("%s %d", s, n)
}
`
	_, out := runGoSource(t, src)
	if out != "9876 1234" {
		t.Fatalf("expected '9876 1234', got '%s'", out)
	}
}

// 14. Go 1.22+ Integer range
func TestRealtimeIntegerRange(t *testing.T) {
	src := `package main
func main() {
	sum := 0
	for i := range 5 {
		sum += i
	}
	fmt.Printf("%d", sum)
}
`
	_, out := runGoSource(t, src)
	if out != "10" {
		t.Fatalf("expected '10', got '%s'", out)
	}
}

// 15. Escape sequences in string literals
func TestRealtimeStringEscapes(t *testing.T) {
	src := `package main
func main() {
	s := "Line1\nLine2\tTabbed\"Quote\""
	fmt.Print(s)
}
`
	_, out := runGoSource(t, src)
	expected := "Line1\nLine2\tTabbed\"Quote\""
	if out != expected {
		t.Fatalf("expected '%s', got '%s'", expected, out)
	}
}
