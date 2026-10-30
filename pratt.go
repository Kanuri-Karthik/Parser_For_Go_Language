package main

import "fmt"

// binaryPrecedence returns the left and right binding powers for a binary operator.
// Go's 5 precedence levels (10 to 50):
// Level 1 (10): ||
// Level 2 (20): &&
// Level 3 (30): == != < <= > >=
// Level 4 (40): + - | ^
// Level 5 (50): * / % << >> & &^
// Left associativity: rightBP = leftBP + 1.
func binaryPrecedence(tok TokenType) (int, int) {
	switch tok {
	case LOR:
		return 10, 11
	case LAND:
		return 20, 21
	case EQL, NEQ, LSS, LEQ, GTR, GEQ:
		return 30, 31
	case ADD, SUB, OR, XOR:
		return 40, 41
	case MUL, QUO, REM, SHL, SHR, AND, AND_NOT:
		return 50, 51
	default:
		return -1, -1
	}
}

const (
	unaryBP   = 60
	postfixBP = 70
)

// parseExpr parses an expression using Pratt parsing with the given minimum binding power.
func (p *Parser) parseExpr(minBP int) Expr {
	tok := p.cur()

	var left Expr

	// 1. Prefix expressions (nud)
	switch tok.Type {
	case IDENT:
		p.advance()
		left = &Ident{Name: tok.Lexeme, pos: tok.Pos}

	case INT, FLOAT, STRING:
		p.advance()
		left = &BasicLit{Kind: tok.Type, Value: tok.Lexeme, pos: tok.Pos}

	case ADD, SUB, NOT, XOR, MUL, AND:
		// Unary prefix operator
		p.advance()
		rhs := p.parseExpr(unaryBP)
		left = &UnaryExpr{
			Op:    tok.Type,
			OpStr: tok.Lexeme,
			X:     rhs,
			pos:   tok.Pos,
		}

	case LPAREN:
		lparen := tok.Pos
		p.advance()
		errsBefore := len(p.errors)
		inner := p.parseExpr(0)
		if p.cur().Type != RPAREN {
			if len(p.errors) == errsBefore || (p.cur().Type != SEMICOLON && p.cur().Type != EOF) {
				p.errorExpected("')'")
			}
		} else {
			p.advance()
		}
		left = &ParenExpr{
			X:   inner,
			pos: lparen,
		}

	case LBRACK:
		// Slice / Array type prefix: e.g. []int, [5]int
		lbrackPos := tok.Pos
		p.advance()
		var lenExpr Expr
		if p.cur().Type != RBRACK {
			p.noCompositeLit = true
			lenExpr = p.parseExpr(0)
			p.noCompositeLit = false
		}
		if p.cur().Type == RBRACK {
			p.advance()
		} else {
			p.errorExpected("']'")
		}
		p.noCompositeLit = true
		elt := p.parseExpr(postfixBP)
		p.noCompositeLit = false
		left = &ArrayType{
			Len: lenExpr,
			Elt: elt,
			pos: lbrackPos,
		}

	case MAP:
		mapTok := p.advance()
		if p.cur().Type != LBRACK {
			p.errorExpected("'['")
		} else {
			p.advance()
		}
		keyExpr := p.parseExpr(0)
		if p.cur().Type != RBRACK {
			p.errorExpected("']'")
		} else {
			p.advance()
		}
		p.noCompositeLit = true
		valExpr := p.parseExpr(postfixBP)
		p.noCompositeLit = false
		left = &MapType{
			Key:   keyExpr,
			Value: valExpr,
			pos:   mapTok.Pos,
		}

	default:
		if tok.Type == ILLEGAL {
			diag := Diagnostic{
				Pos:     tok.Pos,
				Length:  tokenLength(tok),
				Message: fmt.Sprintf("illegal character '%s'", tok.Lexeme),
				Found:   fmt.Sprintf("'%s'", tok.Lexeme),
			}
			p.errors = append(p.errors, diag)
			p.advance()
			return &BadExpr{pos: tok.Pos}
		}
		p.errorExpected("expression")
		return &BadExpr{pos: tok.Pos}
	}

	// 2. Infix and Postfix expressions (led)
	for {
		cur := p.cur()

		// Composite literal: postfix '{'
		if cur.Type == LBRACE && !p.noCompositeLit {
			switch left.(type) {
			case *Ident, *SelectorExpr, *ArrayType, *MapType:
				if postfixBP < minBP {
					break
				}
				p.advance() // consume '{'
				var elts []Expr
				for p.cur().Type != RBRACE && p.cur().Type != EOF {
					if p.cur().Type == SEMICOLON {
						p.advance()
						continue
					}
					elt := p.parseExpr(0)
					if p.cur().Type == COLON {
						p.advance() // consume ':'
						val := p.parseExpr(0)
						elt = &KeyValueExpr{Key: elt, Value: val, pos: elt.Pos()}
					}
					elts = append(elts, elt)
					if p.cur().Type == COMMA {
						p.advance()
					} else {
						break
					}
				}
				if p.cur().Type == RBRACE {
					p.advance()
				} else {
					p.errorExpected("'}'")
				}
				left = &CompositeLit{
					Type: left,
					Elts: elts,
					pos:  left.Pos(),
				}
				continue
			}
		}

		// Function call: postfix '('
		if cur.Type == LPAREN {
			if postfixBP < minBP {
				break
			}
			p.advance()
			var args []Expr
			if p.cur().Type != RPAREN {
				for {
					arg := p.parseExpr(0)
					args = append(args, arg)
					if p.cur().Type == COMMA {
						p.advance()
						if p.cur().Type == RPAREN {
							break
						}
					} else {
						break
					}
				}
			}
			if p.cur().Type != RPAREN {
				p.errorExpected("')'")
			} else {
				p.advance()
			}
			left = &CallExpr{
				Fun:  left,
				Args: args,
				pos:  left.Pos(),
			}
			continue
		}

		// Index / Slice: postfix '['
		if cur.Type == LBRACK {
			if postfixBP < minBP {
				break
			}
			p.advance() // consume '['
			var low Expr
			var high Expr
			var max Expr
			isSlice := false

			if p.cur().Type == COLON {
				isSlice = true
				p.advance()
				if p.cur().Type != COLON && p.cur().Type != RBRACK {
					high = p.parseExpr(0)
				}
				if p.cur().Type == COLON {
					p.advance()
					max = p.parseExpr(0)
				}
			} else if p.cur().Type != RBRACK {
				low = p.parseExpr(0)
				if p.cur().Type == COLON {
					isSlice = true
					p.advance()
					if p.cur().Type != COLON && p.cur().Type != RBRACK {
						high = p.parseExpr(0)
					}
					if p.cur().Type == COLON {
						p.advance()
						max = p.parseExpr(0)
					}
				}
			}

			if p.cur().Type != RBRACK {
				p.errorExpected("']'")
			} else {
				p.advance()
			}

			if isSlice {
				left = &SliceExpr{
					X:    left,
					Low:  low,
					High: high,
					Max:  max,
					pos:  left.Pos(),
				}
			} else {
				left = &IndexExpr{
					X:     left,
					Index: low,
					pos:   left.Pos(),
				}
			}
			continue
		}

		// Selector: postfix '.'
		if cur.Type == PERIOD {
			if postfixBP < minBP {
				break
			}
			p.advance()
			selTok := p.cur()
			if selTok.Type != IDENT {
				p.errorExpected("identifier")
				break
			}
			p.advance()
			left = &SelectorExpr{
				X:   left,
				Sel: &Ident{Name: selTok.Lexeme, pos: selTok.Pos},
				pos: left.Pos(),
			}
			continue
		}

		// Binary operator
		leftBP, rightBP := binaryPrecedence(cur.Type)
		if leftBP == -1 || leftBP < minBP {
			break
		}

		opTok := cur
		p.advance()
		rhs := p.parseExpr(rightBP)

		left = &BinaryExpr{
			Op:    opTok.Type,
			OpStr: opTok.Lexeme,
			X:     left,
			Y:     rhs,
			pos:   opTok.Pos,
		}
	}

	return left
}
