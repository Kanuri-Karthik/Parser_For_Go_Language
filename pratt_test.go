package main

import (
	"testing"
)

func parseExprFrom(src string) (Expr, []Diagnostic) {
	p := NewParser(src)
	expr := p.parseExpr(0)
	return expr, p.errors
}

func TestPrattPrecedenceLevels(t *testing.T) {
	t.Run("mul binds tighter than add: a + b * 2", func(t *testing.T) {
		expr, errs := parseExprFrom("a + b * 2")
		if len(errs) > 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		bin, ok := expr.(*BinaryExpr)
		if !ok || bin.Op != ADD {
			t.Fatalf("expected root to be +, got %T", expr)
		}
		rhs, ok := bin.Y.(*BinaryExpr)
		if !ok || rhs.Op != MUL {
			t.Fatalf("expected rhs to be *, got %T", bin.Y)
		}
	})

	t.Run("add binds tighter than comparison: a == b + c", func(t *testing.T) {
		expr, errs := parseExprFrom("a == b + c")
		if len(errs) > 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		bin, ok := expr.(*BinaryExpr)
		if !ok || bin.Op != EQL {
			t.Fatalf("expected root to be ==, got %T", expr)
		}
		rhs, ok := bin.Y.(*BinaryExpr)
		if !ok || rhs.Op != ADD {
			t.Fatalf("expected rhs to be +, got %T", bin.Y)
		}
	})

	t.Run("comparison binds tighter than and: a && b == c", func(t *testing.T) {
		expr, errs := parseExprFrom("a && b == c")
		if len(errs) > 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		bin, ok := expr.(*BinaryExpr)
		if !ok || bin.Op != LAND {
			t.Fatalf("expected root to be &&, got %T", expr)
		}
		rhs, ok := bin.Y.(*BinaryExpr)
		if !ok || rhs.Op != EQL {
			t.Fatalf("expected rhs to be ==, got %T", bin.Y)
		}
	})

	t.Run("and binds tighter than or: a || b && c", func(t *testing.T) {
		expr, errs := parseExprFrom("a || b && c")
		if len(errs) > 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		bin, ok := expr.(*BinaryExpr)
		if !ok || bin.Op != LOR {
			t.Fatalf("expected root to be ||, got %T", expr)
		}
		rhs, ok := bin.Y.(*BinaryExpr)
		if !ok || rhs.Op != LAND {
			t.Fatalf("expected rhs to be &&, got %T", bin.Y)
		}
	})
}

func TestPrattLeftAssociativity(t *testing.T) {
	t.Run("subtraction left associative: 10 - 4 - 3", func(t *testing.T) {
		expr, errs := parseExprFrom("10 - 4 - 3")
		if len(errs) > 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		// Expect ((10 - 4) - 3)
		root, ok := expr.(*BinaryExpr)
		if !ok || root.Op != SUB {
			t.Fatalf("expected root to be -, got %T", expr)
		}
		lhs, ok := root.X.(*BinaryExpr)
		if !ok || lhs.Op != SUB {
			t.Fatalf("expected lhs to be -, got %T", root.X)
		}
		rhs, ok := root.Y.(*BasicLit)
		if !ok || rhs.Value != "3" {
			t.Fatalf("expected rhs to be 3, got %v", root.Y)
		}
	})
}

func TestPrattUnaryAndParentheses(t *testing.T) {
	t.Run("unary minus binds tighter than mul: -a * b", func(t *testing.T) {
		expr, errs := parseExprFrom("-a * b")
		if len(errs) > 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		root, ok := expr.(*BinaryExpr)
		if !ok || root.Op != MUL {
			t.Fatalf("expected root to be *, got %T", expr)
		}
		_, ok = root.X.(*UnaryExpr)
		if !ok {
			t.Fatalf("expected lhs to be UnaryExpr, got %T", root.X)
		}
	})

	t.Run("parenthesized expression matches Slide 5b: (a + b) * c", func(t *testing.T) {
		expr, errs := parseExprFrom("(a + b) * c")
		if len(errs) > 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		root, ok := expr.(*BinaryExpr)
		if !ok || root.Op != MUL {
			t.Fatalf("expected root to be *, got %T", expr)
		}
		paren, ok := root.X.(*ParenExpr)
		if !ok {
			t.Fatalf("expected lhs to be ParenExpr, got %T", root.X)
		}
		inner, ok := paren.X.(*BinaryExpr)
		if !ok || inner.Op != ADD {
			t.Fatalf("expected inner to be +, got %T", paren.X)
		}
	})
}

func TestPrattCallIndexSelector(t *testing.T) {
	t.Run("call expression: add(2, 3) * 4", func(t *testing.T) {
		expr, errs := parseExprFrom("add(2, 3) * 4")
		if len(errs) > 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		root, ok := expr.(*BinaryExpr)
		if !ok || root.Op != MUL {
			t.Fatalf("expected root *, got %T", expr)
		}
		call, ok := root.X.(*CallExpr)
		if !ok {
			t.Fatalf("expected lhs CallExpr, got %T", root.X)
		}
		if len(call.Args) != 2 {
			t.Fatalf("expected 2 args, got %d", len(call.Args))
		}
	})

	t.Run("selector expression: obj.field + 1", func(t *testing.T) {
		expr, errs := parseExprFrom("obj.field + 1")
		if len(errs) > 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		root, ok := expr.(*BinaryExpr)
		if !ok || root.Op != ADD {
			t.Fatalf("expected root +, got %T", expr)
		}
		sel, ok := root.X.(*SelectorExpr)
		if !ok {
			t.Fatalf("expected lhs SelectorExpr, got %T", root.X)
		}
		if sel.Sel.Name != "field" {
			t.Errorf("expected field, got %s", sel.Sel.Name)
		}
	})
}
