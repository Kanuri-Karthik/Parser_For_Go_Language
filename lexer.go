package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenType represents a lexical token in the Go language subset.
type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF
	COMMENT

	// Identifiers and literals
	IDENT  // x, calc, int
	INT    // 123
	FLOAT  // 123.45
	STRING // "hello"

	// Operators and punctuation
	ADD // +
	SUB // -
	MUL // *
	QUO // /
	REM // %

	AND     // &
	OR      // |
	XOR     // ^
	SHL     // <<
	SHR     // >>
	AND_NOT // &^

	ADD_ASSIGN // +=
	SUB_ASSIGN // -=
	MUL_ASSIGN // *=
	QUO_ASSIGN // /=
	AND_ASSIGN // &=
	OR_ASSIGN  // |=
	XOR_ASSIGN // ^=
	SHL_ASSIGN // <<=
	SHR_ASSIGN // >>=
	AND_NOT_ASSIGN // &^=

	ELLIPSIS // ...
	ARROW    // <-

	LAND // &&
	LOR  // ||
	NOT  // !

	EQL // ==
	LSS // <
	GTR // >
	ASSIGN // =

	NEQ // !=
	LEQ // <=
	GEQ // >=
	DEFINE // :=

	INC // ++
	DEC // --

	LPAREN // (
	LBRACK // [
	LBRACE // {
	COMMA  // ,
	PERIOD // .

	RPAREN    // )
	RBRACK    // ]
	RBRACE    // }
	SEMICOLON // ;
	COLON     // :

	// Keywords (all 25 Go keywords)
	PACKAGE
	IMPORT
	VAR
	CONST
	TYPE
	FUNC
	IF
	ELSE
	FOR
	RETURN
	BREAK
	CONTINUE
	STRUCT
	SWITCH
	CASE
	DEFAULT
	RANGE
	DEFER
	GO
	CHAN
	MAP
	INTERFACE
	SELECT
	GOTO
	FALLTHROUGH
)

var tokenNames = [...]string{
	ILLEGAL:   "ILLEGAL",
	EOF:       "EOF",
	COMMENT:   "COMMENT",

	IDENT:     "IDENT",
	INT:       "INT",
	FLOAT:     "FLOAT",
	STRING:    "STRING",

	ADD:       "ADD",
	SUB:       "SUB",
	MUL:       "MUL",
	QUO:       "QUO",
	REM:       "REM",

	AND:       "AND",
	OR:        "OR",
	XOR:       "XOR",
	SHL:       "SHL",
	SHR:       "SHR",
	AND_NOT:   "AND_NOT",

	ADD_ASSIGN:     "+=",
	SUB_ASSIGN:     "-=",
	MUL_ASSIGN:     "*=",
	QUO_ASSIGN:     "/=",
	AND_ASSIGN:     "&=",
	OR_ASSIGN:      "|=",
	XOR_ASSIGN:     "^=",
	SHL_ASSIGN:     "<<=",
	SHR_ASSIGN:     ">>=",
	AND_NOT_ASSIGN: "&^=",

	ELLIPSIS: "...",
	ARROW:    "<-",

	LAND:      "LAND",
	LOR:       "LOR",
	NOT:       "NOT",

	EQL:       "==",
	LSS:       "<",
	GTR:       ">",
	ASSIGN:    "=",

	NEQ:       "!=",
	LEQ:       "<=",
	GEQ:       ">=",
	DEFINE:    "DEFINE",

	INC:       "++",
	DEC:       "--",

	LPAREN:    "(",
	LBRACK:    "[",
	LBRACE:    "{",
	COMMA:     ",",
	PERIOD:    ".",

	RPAREN:    ")",
	RBRACK:    "]",
	RBRACE:    "}",
	SEMICOLON: "SEMICOLON",
	COLON:     ":",

	PACKAGE:     "PACKAGE",
	IMPORT:      "IMPORT",
	VAR:         "VAR",
	CONST:       "CONST",
	TYPE:        "TYPE",
	FUNC:        "FUNC",
	IF:          "IF",
	ELSE:        "ELSE",
	FOR:         "FOR",
	RETURN:      "RETURN",
	BREAK:       "BREAK",
	CONTINUE:    "CONTINUE",
	STRUCT:      "STRUCT",
	SWITCH:      "SWITCH",
	CASE:        "CASE",
	DEFAULT:     "DEFAULT",
	RANGE:       "RANGE",
	DEFER:       "DEFER",
	GO:          "GO",
	CHAN:        "CHAN",
	MAP:         "MAP",
	INTERFACE:   "INTERFACE",
	SELECT:      "SELECT",
	GOTO:        "GOTO",
	FALLTHROUGH: "FALLTHROUGH",
}

func (t TokenType) String() string {
	if int(t) >= 0 && int(t) < len(tokenNames) {
		return tokenNames[t]
	}
	return fmt.Sprintf("Token(%d)", t)
}

var keywords = map[string]TokenType{
	"package":     PACKAGE,
	"import":      IMPORT,
	"var":         VAR,
	"const":       CONST,
	"type":        TYPE,
	"func":        FUNC,
	"if":          IF,
	"else":        ELSE,
	"for":         FOR,
	"return":      RETURN,
	"break":       BREAK,
	"continue":    CONTINUE,
	"struct":      STRUCT,
	"switch":      SWITCH,
	"case":        CASE,
	"default":     DEFAULT,
	"range":       RANGE,
	"defer":       DEFER,
	"go":          GO,
	"chan":        CHAN,
	"map":         MAP,
	"interface":   INTERFACE,
	"select":      SELECT,
	"goto":        GOTO,
	"fallthrough": FALLTHROUGH,
}

// Position represents a source position.
type Position struct {
	Line int
	Col  int
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}

// Token represents a scanned token.
type Token struct {
	Type       TokenType
	Lexeme     string
	Pos        Position
	IsAutoSemi bool
}

func (t Token) String() string {
	lex := t.Lexeme
	if t.IsAutoSemi {
		lex = `\n auto`
	} else if t.Type == EOF {
		lex = "—"
	}
	return fmt.Sprintf("%-11s %-9s %s", t.Type, lex, t.Pos)
}

// Lexer performs single-pass lexical analysis with automatic semicolon insertion.
type Lexer struct {
	src        string
	offset     int
	ch         rune
	line       int
	col        int
	insertSemi bool
}

func NewLexer(src string) *Lexer {
	l := &Lexer{
		src:    src,
		offset: 0,
		line:   1,
		col:    0,
	}
	l.next()
	return l
}

func (l *Lexer) next() rune {
	if l.offset >= len(l.src) {
		l.ch = -1
		l.offset = len(l.src) + 1
		return -1
	}

	r, size := utf8.DecodeRuneInString(l.src[l.offset:])
	l.offset += size
	l.ch = r
	if r == '\n' {
		l.line++
		l.col = 0
	} else {
		l.col++
	}
	return l.ch
}

func (l *Lexer) peek() rune {
	if l.offset >= len(l.src) {
		return -1
	}
	r, _ := utf8.DecodeRuneInString(l.src[l.offset:])
	return r
}

func (l *Lexer) peekAt(offset int) rune {
	if offset >= len(l.src) {
		return -1
	}
	r, _ := utf8.DecodeRuneInString(l.src[offset:])
	return r
}

func (l *Lexer) NextToken() Token {
	for {
		// Handle automatic semicolon insertion before skipping newlines
		if l.ch == '\n' {
			pos := Position{Line: l.line - 1, Col: l.prevLineLen() + 1}
			l.next()
			if l.insertSemi {
				l.insertSemi = false
				return Token{
					Type:       SEMICOLON,
					Lexeme:     `\n auto`,
					Pos:        pos,
					IsAutoSemi: true,
				}
			}
			continue
		}

		if l.ch == -1 {
			if l.insertSemi {
				l.insertSemi = false
				return Token{
					Type:       SEMICOLON,
					Lexeme:     `\n auto`,
					Pos:        Position{Line: l.line, Col: l.col + 1},
					IsAutoSemi: true,
				}
			}
			eofLine := l.line
			if l.col > 0 {
				eofLine = l.line + 1
			}
			return Token{
				Type:   EOF,
				Lexeme: "EOF",
				Pos:    Position{Line: eofLine, Col: 1},
			}
		}

		// Skip whitespace (excluding newline)
		if unicode.IsSpace(l.ch) {
			l.next()
			continue
		}

		// Handle comments
		if l.ch == '/' {
			if l.peek() == '/' {
				// Line comment
				for l.ch != '\n' && l.ch != -1 {
					l.next()
				}
				continue
			} else if l.peek() == '*' {
				// Block comment
				l.next() // consume /
				l.next() // consume *
				hasNewline := false
				for {
					if l.ch == -1 {
						break
					}
					if l.ch == '\n' {
						hasNewline = true
					}
					if l.ch == '*' && l.peek() == '/' {
						l.next()
						l.next()
						break
					}
					l.next()
				}
				if hasNewline && l.insertSemi {
					l.insertSemi = false
					return Token{
						Type:       SEMICOLON,
						Lexeme:     `\n auto`,
						Pos:        Position{Line: l.line, Col: l.col},
						IsAutoSemi: true,
					}
				}
				continue
			}
		}

		break
	}

	pos := Position{Line: l.line, Col: l.col}

	// Identifiers and keywords
	if isLetter(l.ch) {
		start := l.currentRuneOffset()
		for isLetter(l.ch) || isDigit(l.ch) {
			l.next()
		}
		end := l.currentRuneOffset()
		lit := l.src[start:end]

		if tok, ok := keywords[lit]; ok {
			// Semicolon insertion rule: after return, break, continue, fallthrough
			if tok == RETURN || tok == BREAK || tok == CONTINUE || tok == FALLTHROUGH {
				l.insertSemi = true
			} else {
				l.insertSemi = false
			}
			return Token{Type: tok, Lexeme: lit, Pos: pos}
		}

		l.insertSemi = true
		return Token{Type: IDENT, Lexeme: lit, Pos: pos}
	}

	// Numbers (decimals, hex 0x, binary 0b, octal 0o, floats, scientific notation, and underscores)
	if isDigit(l.ch) {
		start := l.currentRuneOffset()
		isFloat := false

		if l.ch == '0' && (l.peek() == 'x' || l.peek() == 'X') {
			l.next() // consume '0'
			l.next() // consume 'x'
			for isHexDigit(l.ch) || l.ch == '_' {
				l.next()
			}
		} else if l.ch == '0' && (l.peek() == 'b' || l.peek() == 'B') {
			l.next() // consume '0'
			l.next() // consume 'b'
			for l.ch == '0' || l.ch == '1' || l.ch == '_' {
				l.next()
			}
		} else if l.ch == '0' && (l.peek() == 'o' || l.peek() == 'O') {
			l.next() // consume '0'
			l.next() // consume 'o'
			for (l.ch >= '0' && l.ch <= '7') || l.ch == '_' {
				l.next()
			}
		} else {
			for isDigit(l.ch) || l.ch == '_' {
				l.next()
			}
			if l.ch == '.' && isDigit(l.peek()) {
				isFloat = true
				l.next() // consume '.'
				for isDigit(l.ch) || l.ch == '_' {
					l.next()
				}
			}
			if l.ch == 'e' || l.ch == 'E' {
				isFloat = true
				l.next()
				if l.ch == '+' || l.ch == '-' {
					l.next()
				}
				for isDigit(l.ch) || l.ch == '_' {
					l.next()
				}
			}
		}

		end := l.currentRuneOffset()
		lit := l.src[start:end]
		l.insertSemi = true
		if isFloat {
			return Token{Type: FLOAT, Lexeme: lit, Pos: pos}
		}
		return Token{Type: INT, Lexeme: lit, Pos: pos}
	}

	// String literals
	if l.ch == '"' {
		start := l.currentRuneOffset()
		l.next() // consume opening quote
		for l.ch != '"' && l.ch != -1 && l.ch != '\n' {
			if l.ch == '\\' {
				l.next()
			}
			l.next()
		}
		if l.ch == '"' {
			l.next()
		}
		end := l.currentRuneOffset()
		lit := l.src[start:end]
		l.insertSemi = true
		return Token{Type: STRING, Lexeme: lit, Pos: pos}
	}

	// Raw string literals (`...`)
	if l.ch == '`' {
		start := l.currentRuneOffset()
		l.next() // consume opening backtick
		for l.ch != '`' && l.ch != -1 {
			l.next()
		}
		if l.ch == '`' {
			l.next()
		}
		end := l.currentRuneOffset()
		lit := l.src[start:end]
		l.insertSemi = true
		return Token{Type: STRING, Lexeme: lit, Pos: pos}
	}

	// Rune literals ('x')
	if l.ch == '\'' {
		start := l.currentRuneOffset()
		l.next() // consume opening quote
		for l.ch != '\'' && l.ch != -1 && l.ch != '\n' {
			if l.ch == '\\' {
				l.next()
			}
			l.next()
		}
		if l.ch == '\'' {
			l.next()
		}
		end := l.currentRuneOffset()
		lit := l.src[start:end]
		l.insertSemi = true
		return Token{Type: INT, Lexeme: lit, Pos: pos}
	}

	// Operators and punctuation
	ch := l.ch
	l.next()

	switch ch {
	case '+':
		if l.ch == '+' {
			l.next()
			l.insertSemi = true
			return Token{Type: INC, Lexeme: "++", Pos: pos}
		}
		if l.ch == '=' {
			l.next()
			l.insertSemi = false
			return Token{Type: ADD_ASSIGN, Lexeme: "+=", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: ADD, Lexeme: "+", Pos: pos}
	case '-':
		if l.ch == '-' {
			l.next()
			l.insertSemi = true
			return Token{Type: DEC, Lexeme: "--", Pos: pos}
		}
		if l.ch == '=' {
			l.next()
			l.insertSemi = false
			return Token{Type: SUB_ASSIGN, Lexeme: "-=", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: SUB, Lexeme: "-", Pos: pos}
	case '*':
		if l.ch == '=' {
			l.next()
			l.insertSemi = false
			return Token{Type: MUL_ASSIGN, Lexeme: "*=", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: MUL, Lexeme: "*", Pos: pos}
	case '/':
		if l.ch == '=' {
			l.next()
			l.insertSemi = false
			return Token{Type: QUO_ASSIGN, Lexeme: "/=", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: QUO, Lexeme: "/", Pos: pos}
	case '%':
		l.insertSemi = false
		return Token{Type: REM, Lexeme: "%", Pos: pos}
	case '&':
		if l.ch == '&' {
			l.next()
			l.insertSemi = false
			return Token{Type: LAND, Lexeme: "&&", Pos: pos}
		}
		if l.ch == '^' {
			l.next()
			if l.ch == '=' {
				l.next()
				l.insertSemi = false
				return Token{Type: AND_NOT_ASSIGN, Lexeme: "&^=", Pos: pos}
			}
			l.insertSemi = false
			return Token{Type: AND_NOT, Lexeme: "&^", Pos: pos}
		}
		if l.ch == '=' {
			l.next()
			l.insertSemi = false
			return Token{Type: AND_ASSIGN, Lexeme: "&=", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: AND, Lexeme: "&", Pos: pos}
	case '|':
		if l.ch == '|' {
			l.next()
			l.insertSemi = false
			return Token{Type: LOR, Lexeme: "||", Pos: pos}
		}
		if l.ch == '=' {
			l.next()
			l.insertSemi = false
			return Token{Type: OR_ASSIGN, Lexeme: "|=", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: OR, Lexeme: "|", Pos: pos}
	case '^':
		if l.ch == '=' {
			l.next()
			l.insertSemi = false
			return Token{Type: XOR_ASSIGN, Lexeme: "^=", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: XOR, Lexeme: "^", Pos: pos}
	case '<':
		if l.ch == '-' {
			l.next()
			l.insertSemi = false
			return Token{Type: ARROW, Lexeme: "<-", Pos: pos}
		}
		if l.ch == '=' {
			l.next()
			l.insertSemi = false
			return Token{Type: LEQ, Lexeme: "<=", Pos: pos}
		}
		if l.ch == '<' {
			l.next()
			if l.ch == '=' {
				l.next()
				l.insertSemi = false
				return Token{Type: SHL_ASSIGN, Lexeme: "<<=", Pos: pos}
			}
			l.insertSemi = false
			return Token{Type: SHL, Lexeme: "<<", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: LSS, Lexeme: "<", Pos: pos}
	case '>':
		if l.ch == '=' {
			l.next()
			l.insertSemi = false
			return Token{Type: GEQ, Lexeme: ">=", Pos: pos}
		}
		if l.ch == '>' {
			l.next()
			if l.ch == '=' {
				l.next()
				l.insertSemi = false
				return Token{Type: SHR_ASSIGN, Lexeme: ">>=", Pos: pos}
			}
			l.insertSemi = false
			return Token{Type: SHR, Lexeme: ">>", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: GTR, Lexeme: ">", Pos: pos}
	case '=':
		if l.ch == '=' {
			l.next()
			l.insertSemi = false
			return Token{Type: EQL, Lexeme: "==", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: ASSIGN, Lexeme: "=", Pos: pos}
	case '!':
		if l.ch == '=' {
			l.next()
			l.insertSemi = false
			return Token{Type: NEQ, Lexeme: "!=", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: NOT, Lexeme: "!", Pos: pos}
	case ':':
		if l.ch == '=' {
			l.next()
			l.insertSemi = false
			return Token{Type: DEFINE, Lexeme: ":=", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: COLON, Lexeme: ":", Pos: pos}
	case '(':
		l.insertSemi = false
		return Token{Type: LPAREN, Lexeme: "(", Pos: pos}
	case ')':
		l.insertSemi = true
		return Token{Type: RPAREN, Lexeme: ")", Pos: pos}
	case '[':
		l.insertSemi = false
		return Token{Type: LBRACK, Lexeme: "[", Pos: pos}
	case ']':
		l.insertSemi = true
		return Token{Type: RBRACK, Lexeme: "]", Pos: pos}
	case '{':
		l.insertSemi = false
		return Token{Type: LBRACE, Lexeme: "{", Pos: pos}
	case '}':
		l.insertSemi = true
		return Token{Type: RBRACE, Lexeme: "}", Pos: pos}
	case ',':
		l.insertSemi = false
		return Token{Type: COMMA, Lexeme: ",", Pos: pos}
	case '.':
		if l.ch == '.' && l.peek() == '.' {
			l.next()
			l.next()
			l.insertSemi = false
			return Token{Type: ELLIPSIS, Lexeme: "...", Pos: pos}
		}
		l.insertSemi = false
		return Token{Type: PERIOD, Lexeme: ".", Pos: pos}
	case ';':
		l.insertSemi = false
		return Token{Type: SEMICOLON, Lexeme: ";", Pos: pos}
	default:
		l.insertSemi = false
		return Token{Type: ILLEGAL, Lexeme: string(ch), Pos: pos}
	}
}

func (l *Lexer) currentRuneOffset() int {
	if l.offset <= 0 {
		return 0
	}
	if l.ch == -1 {
		return len(l.src)
	}
	_, size := utf8.DecodeLastRuneInString(l.src[:l.offset])
	return l.offset - size
}

func (l *Lexer) prevLineLen() int {
	lines := strings.Split(l.src, "\n")
	prevIdx := l.line - 2
	if prevIdx >= 0 && prevIdx < len(lines) {
		line := strings.TrimRight(lines[prevIdx], "\r")
		return utf8.RuneCountInString(line)
	}
	return 0
}

func isLetter(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}

func isHexDigit(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

// Tokenize returns all tokens until EOF.
func Tokenize(src string) []Token {
	l := NewLexer(src)
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}
	return tokens
}
