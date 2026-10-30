package main

import (
	"strings"
	"testing"
)

func TestEvalSingleExpression(t *testing.T) {
	interp := NewInterpreter()
	interp.GlobalEnv.Define("a", Value{Kind: ValInt, IntVal: 10})
	interp.GlobalEnv.Define("b", Value{Kind: ValInt, IntVal: 5})

	p := NewParser("a + b * 2")
	expr := p.parseExpr(0)
	val, err := interp.Eval(expr, interp.GlobalEnv)
	if err != nil {
		t.Fatalf("unexpected eval error: %v", err)
	}
	// 10 + (5 * 2) = 20
	if val.Kind != ValInt || val.IntVal != 20 {
		t.Errorf("expected 20, got %v", val)
	}
}

func TestEvalSlide4CalcProgram(t *testing.T) {
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

func main() {
	println("Result:", calc(2, 3))
	return calc(2, 3)
}
`
	p := NewParser(src)
	file := p.ParseFile()
	if len(p.errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", p.FormatDiagnostics())
	}

	interp := NewInterpreter()
	res, stdout, err := interp.ExecuteFile(file)
	if err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}

	// For a=2, b=3:
	// x := 2 + 3*2 = 8
	// x = 8 - (10 / 2) = 8 - 5 = 3
	// x > 5 is false
	// return -x -> -3!
	if res.IntVal != -3 {
		t.Errorf("expected return value -3, got %v", res.IntVal)
	}
	if stdout == "" {
		t.Errorf("expected stdout from println")
	}
}

func TestEvalLoopAndCondition(t *testing.T) {
	src := `
total := 0
for i := 1; i <= 5; i = i + 1 {
	total = total + i
}
total
`
	p := NewParser(src)
	file := p.ParseFile()

	interp := NewInterpreter()
	val, _, err := interp.ExecuteFile(file)
	if err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}

	// 1 + 2 + 3 + 4 + 5 = 15
	tot, ok := interp.GlobalEnv.Get("total")
	if !ok || tot.IntVal != 15 {
		t.Errorf("expected total 15, got %v (last val: %v)", tot, val)
	}
}

func TestEvalDynamicScanProgram(t *testing.T) {
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
	cases := []struct {
		input       string
		expectedEvenOdd string
		expectedSum string
	}{
		{"5", "5 is odd", "Sum: 15"},
		{"6", "6 is even", "Sum: 21"},
		{"10", "10 is even", "Sum: 55"},
	}

	for _, tc := range cases {
		p := NewParser(src)
		file := p.ParseFile()
		if len(p.errors) > 0 {
			t.Fatalf("unexpected parse errors: %v", p.FormatDiagnostics())
		}

		interp := NewInterpreterWithIO(strings.NewReader(tc.input), nil)
		_, stdout, err := interp.ExecuteFile(file)
		if err != nil {
			t.Fatalf("unexpected exec error for input %s: %v", tc.input, err)
		}

		if !strings.Contains(stdout, tc.expectedEvenOdd) {
			t.Errorf("for input %s, expected output to contain %q, got: %s", tc.input, tc.expectedEvenOdd, stdout)
		}
		if !strings.Contains(stdout, tc.expectedSum) {
			t.Errorf("for input %s, expected output to contain %q, got: %s", tc.input, tc.expectedSum, stdout)
		}
	}
}

func TestEvalPrimeNumberProgramWithBreakAndBooleans(t *testing.T) {
	src := `package main

import "fmt"

func main() {
	var n int
	fmt.Scan(&n)

	prime := true

	if n < 2 {
		prime = false
	}

	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			prime = false
			break
		}
	}

	if prime {
		fmt.Println(n, "is prime")
	} else {
		fmt.Println(n, "is not prime")
	}
}
`
	cases := []struct {
		input    string
		expected string
	}{
		{"5", "5 is prime"},
		{"4", "4 is not prime"},
		{"7", "7 is prime"},
		{"9", "9 is not prime"},
		{"1", "1 is not prime"},
		{"13", "13 is prime"},
	}

	for _, tc := range cases {
		p := NewParser(src)
		file := p.ParseFile()
		if len(p.errors) > 0 {
			t.Fatalf("unexpected parse errors: %v", p.FormatDiagnostics())
		}

		interp := NewInterpreterWithIO(strings.NewReader(tc.input), nil)
		_, stdout, err := interp.ExecuteFile(file)
		if err != nil {
			t.Fatalf("unexpected exec error for input %s: %v", tc.input, err)
		}

		if !strings.Contains(stdout, tc.expected) {
			t.Errorf("for input %s, expected output to contain %q, got: %s", tc.input, tc.expected, stdout)
		}
	}
}
