package main

import (
	"fmt"
	"strings"
)

// Diagnostic represents a compilation error with exact source position.
type Diagnostic struct {
	Pos     Position
	Length  int
	Message string
	Found   string
}

func (d Diagnostic) Format(srcLines []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("error[%d:%d]: %s\n", d.Pos.Line, d.Pos.Col, d.Message))

	lineIdx := d.Pos.Line - 1
	lineContent := ""
	if lineIdx >= 0 && lineIdx < len(srcLines) {
		lineContent = srcLines[lineIdx]
	}

	lineNumStr := fmt.Sprintf("%d", d.Pos.Line)
	sb.WriteString(fmt.Sprintf("%s | %s\n", lineNumStr, lineContent))

	spacesIndent := strings.Repeat(" ", len(lineNumStr))
	caretCol := d.Pos.Col - 1
	if caretCol < 0 {
		caretCol = 0
	}
	runes := []rune(lineContent)
	var caretIndent strings.Builder
	for i := 0; i < len(runes) && i < caretCol; i++ {
		if runes[i] == '\t' {
			caretIndent.WriteRune('\t')
		} else {
			caretIndent.WriteRune(' ')
		}
	}
	if caretCol > len(runes) {
		caretIndent.WriteString(strings.Repeat(" ", caretCol-len(runes)))
	}
	length := d.Length
	if length <= 0 {
		length = 1
	}
	carets := strings.Repeat("^", length)
	if d.Found != "" {
		sb.WriteString(fmt.Sprintf("%s | %s%s found %s", spacesIndent, caretIndent.String(), carets, d.Found))
	} else {
		sb.WriteString(fmt.Sprintf("%s | %s%s", spacesIndent, caretIndent.String(), carets))
	}
	return sb.String()
}

// Parser parses Go tokens into an AST using Pratt parsing and recursive descent.
type Parser struct {
	tokens         []Token
	pos            int
	errors         []Diagnostic
	src            string
	srcLines       []string
	noCompositeLit bool
}

func NewParser(src string) *Parser {
	tokens := Tokenize(src)
	lines := strings.Split(src, "\n")
	return &Parser{
		tokens:   tokens,
		pos:      0,
		src:      src,
		srcLines: lines,
	}
}

func (p *Parser) cur() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return Token{Type: EOF, Lexeme: "EOF", Pos: Position{Line: len(p.srcLines), Col: 1}}
}

func (p *Parser) peek() Token {
	if p.pos+1 < len(p.tokens) {
		return p.tokens[p.pos+1]
	}
	return Token{Type: EOF, Lexeme: "EOF", Pos: Position{Line: len(p.srcLines), Col: 1}}
}

func (p *Parser) advance() Token {
	tok := p.cur()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) errorExpected(expected string) {
	tok := p.cur()
	length := tokenLength(tok)
	found := tokenFoundDesc(tok)

	diag := Diagnostic{
		Pos:     tok.Pos,
		Length:  length,
		Message: "expected " + expected,
		Found:   found,
	}
	p.errors = append(p.errors, diag)
}

func tokenLength(tok Token) int {
	if tok.Type == EOF {
		return 1
	}
	if tok.IsAutoSemi {
		return 1
	}
	if tok.Type == INT || tok.Type == FLOAT || tok.Type == STRING || tok.Type == IDENT {
		return len(tok.Lexeme)
	}
	if tok.Type >= PACKAGE && tok.Type <= FALLTHROUGH {
		return len(tok.Lexeme)
	}
	if len(tok.Lexeme) > 0 {
		return len(tok.Lexeme)
	}
	return 1
}

func tokenFoundDesc(tok Token) string {
	switch tok.Type {
	case INT:
		return fmt.Sprintf("INT(%s)", tok.Lexeme)
	case FLOAT:
		return fmt.Sprintf("FLOAT(%s)", tok.Lexeme)
	case STRING:
		return fmt.Sprintf("%q", tok.Lexeme)
	case IDENT:
		return fmt.Sprintf("IDENT(%s)", tok.Lexeme)
	case EOF:
		return "EOF"
	case SEMICOLON:
		return "';'"
	default:
		if tok.Type >= PACKAGE && tok.Type <= FALLTHROUGH {
			return tok.Type.String()
		}
		return fmt.Sprintf("'%s'", tok.Lexeme)
	}
}

// synchronize implements panic-mode recovery by skipping tokens
// until reaching a statement/declaration boundary or semicolon/brace.
func (p *Parser) synchronize() {
	for {
		cur := p.cur()
		if cur.Type == EOF {
			return
		}
		if cur.Type == SEMICOLON {
			p.advance() // consume the semicolon to start fresh
			return
		}
		// If at a boundary keyword, stop without consuming so the parser can resume
		switch cur.Type {
		case PACKAGE, IMPORT, VAR, CONST, TYPE, FUNC, IF, FOR, RETURN, BREAK, CONTINUE, STRUCT, SWITCH, CASE, DEFAULT, RANGE, DEFER, GO, SELECT, RBRACE:
			return
		}
		p.advance()
	}
}

// ParseFile parses a complete Go file (or snippet).
func (p *Parser) ParseFile() *File {
	file := &File{
		pos: p.cur().Pos,
	}

	// Optional package declaration
	if p.cur().Type == PACKAGE {
		pkgPos := p.cur().Pos
		file.PackagePos = pkgPos
		p.advance()
		if p.cur().Type == IDENT {
			file.Name = &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos}
			p.advance()
		} else {
			p.errorExpected("identifier")
		}
		if p.cur().Type == SEMICOLON {
			p.advance()
		}
	}

	for p.cur().Type != EOF {
		startPos := p.pos
		if p.cur().Type == SEMICOLON {
			p.advance()
			continue
		}

		errCountBefore := len(p.errors)

		switch p.cur().Type {
		case IMPORT:
			decl := p.parseImportDecl()
			file.Decls = append(file.Decls, decl)
		case CONST:
			decl := p.parseConstDecl()
			file.Decls = append(file.Decls, decl)
		case TYPE:
			decl := p.parseTypeDecl()
			file.Decls = append(file.Decls, decl)
		case VAR:
			decl := p.parseVarDecl()
			file.Decls = append(file.Decls, decl)
		case FUNC:
			decl := p.parseFuncDecl()
			file.Decls = append(file.Decls, decl)
		default:
			// Parse as top-level statement (support Go snippets)
			stmt := p.parseStmt()
			file.Stmts = append(file.Stmts, stmt)
		}

		if len(p.errors) > errCountBefore {
			p.synchronize()
		} else if p.cur().Type == SEMICOLON {
			p.advance()
		} else if p.cur().Type != EOF {
			p.errorExpected("';'")
			p.synchronize()
		}

		if p.pos <= startPos {
			p.advance() // Guarantee forward progress
		}
	}

	return file
}

func (p *Parser) parseImportDecl() *ImportDecl {
	importTok := p.advance() // consume 'import'
	var specs []*ImportSpec

	if p.cur().Type == LPAREN {
		p.advance() // consume '('
		for p.cur().Type != RPAREN && p.cur().Type != EOF {
			if p.cur().Type == SEMICOLON {
				p.advance()
				continue
			}
			spec := p.parseImportSpec()
			if spec != nil {
				specs = append(specs, spec)
			}
			if p.cur().Type == SEMICOLON {
				p.advance()
			}
		}
		if p.cur().Type == RPAREN {
			p.advance()
		} else {
			p.errorExpected("')'")
		}
	} else {
		spec := p.parseImportSpec()
		if spec != nil {
			specs = append(specs, spec)
		}
	}

	return &ImportDecl{Specs: specs, pos: importTok.Pos}
}

func (p *Parser) parseImportSpec() *ImportSpec {
	var name *Ident
	if p.cur().Type == IDENT {
		name = &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos}
		p.advance()
	}
	if p.cur().Type == STRING {
		pathTok := p.cur()
		p.advance()
		return &ImportSpec{
			Name: name,
			Path: &BasicLit{Kind: STRING, Value: pathTok.Lexeme, pos: pathTok.Pos},
			pos:  pathTok.Pos,
		}
	}
	p.errorExpected("string literal")
	return nil
}

func (p *Parser) parseVarDecl() *VarDecl {
	varTok := p.advance() // consume 'var'

	if p.cur().Type == LPAREN {
		p.advance() // consume '('
		var group []*VarDecl
		for p.cur().Type != RPAREN && p.cur().Type != EOF {
			if p.cur().Type == SEMICOLON {
				p.advance()
				continue
			}
			sub := p.parseVarSpec(p.cur().Pos)
			if sub != nil {
				group = append(group, sub)
			}
			if p.cur().Type == SEMICOLON {
				p.advance()
			}
		}
		if p.cur().Type == RPAREN {
			p.advance()
		} else {
			p.errorExpected("')'")
		}
		return &VarDecl{Group: group, pos: varTok.Pos}
	}

	return p.parseVarSpec(varTok.Pos)
}

func (p *Parser) parseType() Expr {
	switch p.cur().Type {
	case IDENT:
		id := &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos}
		p.advance()
		if p.cur().Type == PERIOD {
			p.advance()
			if p.cur().Type == IDENT {
				sel := &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos}
				p.advance()
				return &SelectorExpr{X: id, Sel: sel, pos: id.pos}
			}
		}
		return id
	case MUL:
		mulTok := p.advance()
		sub := p.parseType()
		return &UnaryExpr{Op: MUL, OpStr: "*", X: sub, pos: mulTok.Pos}
	case LBRACK:
		lbrackPos := p.cur().Pos
		p.advance()
		var lenExpr Expr
		if p.cur().Type != RBRACK {
			lenExpr = p.parseExpr(0)
		}
		if p.cur().Type == RBRACK {
			p.advance()
		} else {
			p.errorExpected("']'")
		}
		elt := p.parseType()
		return &ArrayType{Len: lenExpr, Elt: elt, pos: lbrackPos}
	case MAP:
		mapPos := p.cur().Pos
		p.advance() // consume 'map'
		if p.cur().Type == LBRACK {
			p.advance()
		} else {
			p.errorExpected("'['")
		}
		key := p.parseType()
		if p.cur().Type == RBRACK {
			p.advance()
		} else {
			p.errorExpected("']'")
		}
		val := p.parseType()
		return &MapType{Key: key, Value: val, pos: mapPos}
	default:
		return nil
	}
}

func (p *Parser) parseVarSpec(pos Position) *VarDecl {
	if p.cur().Type != IDENT {
		p.errorExpected("identifier")
		return &VarDecl{pos: pos}
	}

	name := &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos}
	p.advance()
	names := []*Ident{name}

	for p.cur().Type == COMMA {
		p.advance() // consume ','
		if p.cur().Type != IDENT {
			p.errorExpected("identifier")
			break
		}
		names = append(names, &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos})
		p.advance()
	}

	var typeExpr Expr
	var valExpr Expr
	var valExprs []Expr

	// Optional Type
	if p.cur().Type == IDENT || p.cur().Type == MUL || p.cur().Type == LBRACK || p.cur().Type == MAP {
		typeExpr = p.parseType()
	}

	// Initialization '=' or expected ';'
	if p.cur().Type == ASSIGN {
		p.advance() // consume '='
		valExpr = p.parseExpr(0)
		valExprs = append(valExprs, valExpr)
		for p.cur().Type == COMMA {
			p.advance() // consume ','
			v := p.parseExpr(0)
			valExprs = append(valExprs, v)
		}
	} else if typeExpr == nil {
		p.errorExpected("type or '='")
	} else if p.cur().Type != SEMICOLON && p.cur().Type != EOF {
		// Missing '=' or missing ';' (e.g. "var y int 10")
		p.errorExpected("';'")
	}

	return &VarDecl{
		Name:   name,
		Names:  names,
		Type:   typeExpr,
		Value:  valExpr,
		Values: valExprs,
		pos:    pos,
	}
}

func (p *Parser) parseFuncDecl() *FuncDecl {
	funcTok := p.advance() // consume 'func'

	var recv *Field
	if p.cur().Type == LPAREN {
		p.advance()
		rList := p.parseParamList()
		if len(rList) > 0 {
			recv = rList[0]
		}
	}

	if p.cur().Type != IDENT {
		p.errorExpected("identifier")
		return &FuncDecl{pos: funcTok.Pos}
	}
	name := &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos}
	p.advance()

	if p.cur().Type != LPAREN {
		p.errorExpected("'('")
		return &FuncDecl{Recv: recv, Name: name, pos: funcTok.Pos}
	}
	p.advance()

	// Parameters
	params := p.parseParamList()

	// Return types (optional)
	var retType Expr
	var results []*Field
	if p.cur().Type == LPAREN {
		p.advance()
		results = p.parseParamList()
	} else if p.cur().Type == IDENT || p.cur().Type == MUL || p.cur().Type == LBRACK || p.cur().Type == MAP {
		retType = p.parseType()
	}

	// Body Block
	var body *BlockStmt
	if p.cur().Type != LBRACE {
		p.errorExpected("'{'")
	} else {
		body = p.parseBlock()
	}

	return &FuncDecl{
		Recv:    recv,
		Name:    name,
		Params:  params,
		Type:    retType,
		Results: results,
		Body:    body,
		pos:     funcTok.Pos,
	}
}

func (p *Parser) parseParamList() []*Field {
	var params []*Field
	if p.cur().Type == RPAREN {
		p.advance()
		return params
	}

	// If right after '(', next is '{', then missing ')'
	if p.cur().Type == LBRACE {
		p.errorExpected("')'")
		return params
	}

	for p.cur().Type != RPAREN && p.cur().Type != EOF && p.cur().Type != LBRACE {
		var names []*Ident
		for p.cur().Type == IDENT {
			names = append(names, &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos})
			p.advance()
			if p.cur().Type == COMMA {
				p.advance()
			} else {
				break
			}
		}

		var typeExpr Expr
		if p.cur().Type == IDENT || p.cur().Type == MUL || p.cur().Type == LBRACK || p.cur().Type == MAP {
			typeExpr = p.parseType()
		}

		if len(names) > 0 {
			if typeExpr == nil && len(names) == 1 {
				// Unnamed parameter or return type e.g. (int)
				params = append(params, &Field{Type: names[0]})
			} else {
				params = append(params, &Field{Names: names, Type: typeExpr})
			}
		} else if typeExpr != nil {
			params = append(params, &Field{Type: typeExpr})
		}

		if p.cur().Type == COMMA {
			p.advance()
		} else {
			break
		}
	}

	if p.cur().Type != RPAREN {
		p.errorExpected("')'")
	} else {
		p.advance() // consume ')'
	}

	return params
}

func (p *Parser) parseBlock() *BlockStmt {
	lbrace := p.cur().Pos
	p.advance() // consume '{'

	blockErrCount := len(p.errors)
	var list []Stmt
	for p.cur().Type != RBRACE && p.cur().Type != EOF {
		startPos := p.pos
		if p.cur().Type == SEMICOLON {
			p.advance()
			continue
		}
		errCountBefore := len(p.errors)
		s := p.parseStmt()
		list = append(list, s)
		if len(p.errors) > errCountBefore {
			p.synchronize()
		} else if p.cur().Type == SEMICOLON {
			p.advance()
		} else if p.cur().Type != RBRACE && p.cur().Type != EOF {
			p.errorExpected("';'")
			p.synchronize()
		}
		if p.pos <= startPos {
			p.advance() // Guarantee forward progress
		}
	}

	if p.cur().Type == RBRACE {
		p.advance()
	} else if len(p.errors) == blockErrCount {
		p.errorExpected("'}'")
	}
	return &BlockStmt{List: list, pos: lbrace}
}

func (p *Parser) parseStmt() Stmt {
	tok := p.cur()

	switch tok.Type {
	case LBRACE:
		return p.parseBlock()

	case CONST:
		return p.parseConstDecl()

	case TYPE:
		return p.parseTypeDecl()

	case VAR:
		decl := p.parseVarDecl()
		return decl

	case IF:
		return p.parseIfStmt()

	case FOR:
		return p.parseForStmt()

	case SWITCH:
		return p.parseSwitchStmt()

	case DEFER:
		return p.parseDeferStmt()

	case RETURN:
		retTok := p.advance()
		var results []Expr
		if p.cur().Type != SEMICOLON && p.cur().Type != RBRACE && p.cur().Type != EOF {
			res := p.parseExpr(0)
			results = append(results, res)
			for p.cur().Type == COMMA {
				p.advance()
				results = append(results, p.parseExpr(0))
			}
		}
		return &ReturnStmt{Results: results, pos: retTok.Pos}

	case BREAK, CONTINUE, FALLTHROUGH:
		tok := p.advance()
		return &BranchStmt{Tok: tok.Type, pos: tok.Pos}

	case ILLEGAL:
		diag := Diagnostic{
			Pos:     tok.Pos,
			Length:  tokenLength(tok),
			Message: fmt.Sprintf("illegal character '%s'", tok.Lexeme),
			Found:   fmt.Sprintf("'%s'", tok.Lexeme),
		}
		p.errors = append(p.errors, diag)
		p.advance()
		return &BadStmt{pos: tok.Pos}

	default:
		// Expression, short var declaration (:=), assignment (=), or inc/dec (++)
		expr := p.parseExpr(0)

		if p.cur().Type == INC || p.cur().Type == DEC {
			tok := p.advance()
			return &IncDecStmt{
				X:     expr,
				Tok:   tok.Type,
				OpStr: tok.Lexeme,
				pos:   tok.Pos,
			}
		}

		// Support multi-LHS: a, b := 1, 2 or a, b = b, a
		lhsList := []Expr{expr}
		for p.cur().Type == COMMA {
			p.advance()
			lhsList = append(lhsList, p.parseExpr(0))
		}

		if p.cur().Type == DEFINE {
			p.advance()
			rhsList := []Expr{p.parseExpr(0)}
			for p.cur().Type == COMMA {
				p.advance()
				rhsList = append(rhsList, p.parseExpr(0))
			}
			return &AssignStmt{
				Tok:     DEFINE,
				OpStr:   ":=",
				Lhs:     lhsList[0],
				Rhs:     rhsList[0],
				LhsList: lhsList,
				RhsList: rhsList,
				pos:     expr.Pos(),
			}
		}

		if p.cur().Type == ASSIGN || p.cur().Type == ADD_ASSIGN || p.cur().Type == SUB_ASSIGN ||
			p.cur().Type == MUL_ASSIGN || p.cur().Type == QUO_ASSIGN || p.cur().Type == AND_ASSIGN ||
			p.cur().Type == OR_ASSIGN || p.cur().Type == XOR_ASSIGN || p.cur().Type == SHL_ASSIGN ||
			p.cur().Type == SHR_ASSIGN || p.cur().Type == AND_NOT_ASSIGN {
			asgTok := p.advance()
			rhsList := []Expr{p.parseExpr(0)}
			for p.cur().Type == COMMA {
				p.advance()
				rhsList = append(rhsList, p.parseExpr(0))
			}
			return &AssignStmt{
				Tok:     asgTok.Type,
				OpStr:   asgTok.Lexeme,
				Lhs:     lhsList[0],
				Rhs:     rhsList[0],
				LhsList: lhsList,
				RhsList: rhsList,
				pos:     expr.Pos(),
			}
		}

		if len(lhsList) > 1 {
			p.errorExpected("assignment or declaration")
			return &BadStmt{pos: expr.Pos()}
		}

		return &ExprStmt{X: expr, pos: expr.Pos()}
	}
}

func (p *Parser) parseIfStmt() *IfStmt {
	ifTok := p.advance() // consume 'if'

	var initStmt Stmt
	p.noCompositeLit = true
	first := p.parseExpr(0)
	p.noCompositeLit = false

	var cond Expr
	if p.cur().Type == SEMICOLON {
		p.advance()
		initStmt = &ExprStmt{X: first, pos: first.Pos()}
		p.noCompositeLit = true
		cond = p.parseExpr(0)
		p.noCompositeLit = false
	} else if p.cur().Type == DEFINE || p.cur().Type == ASSIGN {
		asgTok := p.advance()
		rhs := p.parseExpr(0)
		initStmt = &AssignStmt{
			Tok:   asgTok.Type,
			OpStr: asgTok.Lexeme,
			Lhs:   first,
			Rhs:   rhs,
			pos:   first.Pos(),
		}
		if p.cur().Type == SEMICOLON {
			p.advance()
			p.noCompositeLit = true
			cond = p.parseExpr(0)
			p.noCompositeLit = false
		}
	} else {
		cond = first
	}

	var body *BlockStmt
	if p.cur().Type != LBRACE {
		p.errorExpected("'{'")
		return &IfStmt{Init: initStmt, Cond: cond, pos: ifTok.Pos}
	} else {
		body = p.parseBlock()
	}

	var elseStmt Stmt
	if p.cur().Type == ELSE {
		p.advance()
		if p.cur().Type == IF {
			elseStmt = p.parseIfStmt()
		} else if p.cur().Type == LBRACE {
			elseStmt = p.parseBlock()
		} else {
			p.errorExpected("'{'")
		}
	}

	return &IfStmt{
		Init: initStmt,
		Cond: cond,
		Body: body,
		Else: elseStmt,
		pos:  ifTok.Pos,
	}
}

func (p *Parser) parseForStmt() Stmt {
	forTok := p.advance() // consume 'for'

	// If block directly follows: "for { ... }"
	if p.cur().Type == LBRACE {
		body := p.parseBlock()
		return &ForStmt{Body: body, pos: forTok.Pos}
	}

	// Case 1: "for range expr { ... }"
	if p.cur().Type == RANGE {
		p.advance()
		p.noCompositeLit = true
		rangeX := p.parseExpr(0)
		p.noCompositeLit = false
		body := p.parseBlock()
		return &RangeStmt{X: rangeX, Body: body, pos: forTok.Pos}
	}

	p.noCompositeLit = true
	firstExpr := p.parseExpr(0)
	p.noCompositeLit = false

	// Case 2: "for i, v := range expr" or "for i, v = range expr"
	if p.cur().Type == COMMA {
		p.advance() // consume ','
		p.noCompositeLit = true
		secondExpr := p.parseExpr(0)
		p.noCompositeLit = false
		if p.cur().Type == DEFINE || p.cur().Type == ASSIGN {
			asgTok := p.advance()
			if p.cur().Type == RANGE {
				p.advance() // consume 'range'
				p.noCompositeLit = true
				rangeX := p.parseExpr(0)
				p.noCompositeLit = false
				body := p.parseBlock()
				return &RangeStmt{
					Key:   firstExpr,
					Value: secondExpr,
					Tok:   asgTok.Type,
					X:     rangeX,
					Body:  body,
					pos:   forTok.Pos,
				}
			}
		}
	}

	// Case 3: "for i := range expr" or "for i = range expr"
	if p.cur().Type == DEFINE || p.cur().Type == ASSIGN {
		asgTok := p.advance()
		if p.cur().Type == RANGE {
			p.advance() // consume 'range'
			p.noCompositeLit = true
			rangeX := p.parseExpr(0)
			p.noCompositeLit = false
			body := p.parseBlock()
			return &RangeStmt{
				Key:  firstExpr,
				Tok:  asgTok.Type,
				X:    rangeX,
				Body: body,
				pos:  forTok.Pos,
			}
		}
		// Traditional 3-clause for loop init
		rhs := p.parseExpr(0)
		initStmt := &AssignStmt{
			Tok:   asgTok.Type,
			OpStr: asgTok.Lexeme,
			Lhs:   firstExpr,
			Rhs:   rhs,
			pos:   firstExpr.Pos(),
		}
		return p.parseRemainingForLoop(forTok.Pos, initStmt)
	}

	// Case 4: 3-clause for loop starting with expression statement
	if p.cur().Type == SEMICOLON {
		initStmt := &ExprStmt{X: firstExpr, pos: firstExpr.Pos()}
		return p.parseRemainingForLoop(forTok.Pos, initStmt)
	}

	// Case 5: Single condition for loop: "for cond { ... }"
	var body *BlockStmt
	if p.cur().Type != LBRACE {
		p.errorExpected("'{'")
	} else {
		body = p.parseBlock()
	}

	return &ForStmt{
		Cond: firstExpr,
		Body: body,
		pos:  forTok.Pos,
	}
}

func (p *Parser) parseRemainingForLoop(pos Position, initStmt Stmt) *ForStmt {
	var condExpr Expr
	var postStmt Stmt

	if p.cur().Type == SEMICOLON {
		p.advance() // consume ';'
		if p.cur().Type != SEMICOLON && p.cur().Type != LBRACE {
			condExpr = p.parseExpr(0)
		}
		if p.cur().Type == SEMICOLON {
			p.advance()
		} else {
			p.errorExpected("';'")
		}
		if p.cur().Type != LBRACE {
			postExpr := p.parseExpr(0)
			if p.cur().Type == INC || p.cur().Type == DEC {
				tok := p.advance()
				postStmt = &IncDecStmt{
					X:     postExpr,
					Tok:   tok.Type,
					OpStr: tok.Lexeme,
					pos:   tok.Pos,
				}
			} else if p.cur().Type == ASSIGN || p.cur().Type == ADD_ASSIGN || p.cur().Type == SUB_ASSIGN ||
				p.cur().Type == MUL_ASSIGN || p.cur().Type == QUO_ASSIGN || p.cur().Type == AND_ASSIGN ||
				p.cur().Type == OR_ASSIGN || p.cur().Type == XOR_ASSIGN || p.cur().Type == SHL_ASSIGN ||
				p.cur().Type == SHR_ASSIGN || p.cur().Type == AND_NOT_ASSIGN {
				asgTok := p.advance()
				rhs := p.parseExpr(0)
				postStmt = &AssignStmt{
					Tok:   asgTok.Type,
					OpStr: asgTok.Lexeme,
					Lhs:   postExpr,
					Rhs:   rhs,
					pos:   postExpr.Pos(),
				}
			} else {
				postStmt = &ExprStmt{X: postExpr, pos: postExpr.Pos()}
			}
		}
	}

	var body *BlockStmt
	if p.cur().Type != LBRACE {
		p.errorExpected("'{'")
	} else {
		body = p.parseBlock()
	}

	return &ForStmt{
		Init: initStmt,
		Cond: condExpr,
		Post: postStmt,
		Body: body,
		pos:  pos,
	}
}

func (p *Parser) parseTypeDecl() *TypeDecl {
	typeTok := p.advance() // consume 'type'

	if p.cur().Type != IDENT {
		p.errorExpected("identifier")
		return &TypeDecl{pos: typeTok.Pos}
	}
	name := &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos}
	p.advance()

	// Optional '=' for alias
	if p.cur().Type == ASSIGN {
		p.advance()
	}

	if p.cur().Type == STRUCT {
		structTok := p.advance()
		var fields []*Field
		if p.cur().Type != LBRACE {
			p.errorExpected("'{'")
		} else {
			p.advance() // consume '{'
			for p.cur().Type != RBRACE && p.cur().Type != EOF {
				if p.cur().Type == SEMICOLON {
					p.advance()
					continue
				}
				var fNames []*Ident
				for p.cur().Type == IDENT {
					fNames = append(fNames, &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos})
					p.advance()
					if p.cur().Type == COMMA {
						p.advance()
					} else {
						break
					}
				}
				var fType Expr
				if p.cur().Type != SEMICOLON && p.cur().Type != RBRACE && p.cur().Type != EOF {
					fType = p.parseExpr(0)
				}
				// Optional tag
				if p.cur().Type == STRING {
					p.advance()
				}
				if len(fNames) > 0 {
					fields = append(fields, &Field{Names: fNames, Type: fType})
				}
				if p.cur().Type == SEMICOLON {
					p.advance()
				}
			}
			if p.cur().Type == RBRACE {
				p.advance()
			} else {
				p.errorExpected("'}'")
			}
		}
		return &TypeDecl{
			Name: name,
			Type: &StructType{Fields: fields, pos: structTok.Pos},
			pos:  typeTok.Pos,
		}
	}

	typeExpr := p.parseExpr(0)
	return &TypeDecl{
		Name: name,
		Type: typeExpr,
		pos:  typeTok.Pos,
	}
}

func (p *Parser) parseSwitchStmt() *SwitchStmt {
	switchTok := p.advance() // consume 'switch'

	var initStmt Stmt
	var tagExpr Expr

	if p.cur().Type != LBRACE {
		p.noCompositeLit = true
		first := p.parseExpr(0)
		p.noCompositeLit = false

		if p.cur().Type == DEFINE || p.cur().Type == ASSIGN {
			asgTok := p.advance()
			rhs := p.parseExpr(0)
			initStmt = &AssignStmt{
				Tok:   asgTok.Type,
				OpStr: asgTok.Lexeme,
				Lhs:   first,
				Rhs:   rhs,
				pos:   first.Pos(),
			}
			if p.cur().Type == SEMICOLON {
				p.advance()
				if p.cur().Type != LBRACE {
					tagExpr = p.parseExpr(0)
				}
			}
		} else if p.cur().Type == SEMICOLON {
			p.advance()
			initStmt = &ExprStmt{X: first, pos: first.Pos()}
			if p.cur().Type != LBRACE {
				tagExpr = p.parseExpr(0)
			}
		} else {
			tagExpr = first
		}
	}

	var cases []*CaseClause
	if p.cur().Type != LBRACE {
		p.errorExpected("'{'")
		return &SwitchStmt{Init: initStmt, Tag: tagExpr, pos: switchTok.Pos}
	}
	p.advance() // consume '{'

	for p.cur().Type != RBRACE && p.cur().Type != EOF {
		if p.cur().Type == SEMICOLON {
			p.advance()
			continue
		}

		if p.cur().Type == CASE {
			caseTok := p.advance()
			var conds []Expr
			for {
				cond := p.parseExpr(0)
				conds = append(conds, cond)
				if p.cur().Type == COMMA {
					p.advance()
				} else {
					break
				}
			}
			if p.cur().Type == COLON {
				p.advance()
			} else {
				p.errorExpected("':'")
			}
			var body []Stmt
			for p.cur().Type != CASE && p.cur().Type != DEFAULT && p.cur().Type != RBRACE && p.cur().Type != EOF {
				if p.cur().Type == SEMICOLON {
					p.advance()
					continue
				}
				s := p.parseStmt()
				body = append(body, s)
				if p.cur().Type == SEMICOLON {
					p.advance()
				}
			}
			cases = append(cases, &CaseClause{List: conds, Body: body, pos: caseTok.Pos})
		} else if p.cur().Type == DEFAULT {
			defTok := p.advance()
			if p.cur().Type == COLON {
				p.advance()
			} else {
				p.errorExpected("':'")
			}
			var body []Stmt
			for p.cur().Type != CASE && p.cur().Type != DEFAULT && p.cur().Type != RBRACE && p.cur().Type != EOF {
				if p.cur().Type == SEMICOLON {
					p.advance()
					continue
				}
				s := p.parseStmt()
				body = append(body, s)
				if p.cur().Type == SEMICOLON {
					p.advance()
				}
			}
			cases = append(cases, &CaseClause{List: nil, Body: body, pos: defTok.Pos})
		} else {
			p.errorExpected("'case' or 'default'")
			p.synchronize()
		}
	}

	if p.cur().Type == RBRACE {
		p.advance()
	} else {
		p.errorExpected("'}'")
	}

	return &SwitchStmt{
		Init: initStmt,
		Tag:  tagExpr,
		Body: cases,
		pos:  switchTok.Pos,
	}
}

func (p *Parser) parseDeferStmt() *DeferStmt {
	deferTok := p.advance() // consume 'defer'
	expr := p.parseExpr(0)
	var call *CallExpr
	if c, ok := expr.(*CallExpr); ok {
		call = c
	} else {
		call = &CallExpr{Fun: expr, pos: expr.Pos()}
	}
	return &DeferStmt{Call: call, pos: deferTok.Pos}
}

func (p *Parser) parseConstDecl() *ConstDecl {
	constTok := p.advance() // consume 'const'

	if p.cur().Type == LPAREN {
		p.advance() // consume '('
		var group []*ConstDecl
		var lastValues []Expr
		var lastType Expr
		for p.cur().Type != RPAREN && p.cur().Type != EOF {
			if p.cur().Type == SEMICOLON {
				p.advance()
				continue
			}
			sub := p.parseConstSpecInGroup(p.cur().Pos, lastType, lastValues)
			if sub != nil {
				group = append(group, sub)
				lastValues = sub.Values
				lastType = sub.Type
			}
			if p.cur().Type == SEMICOLON {
				p.advance()
			}
		}
		if p.cur().Type == RPAREN {
			p.advance()
		} else {
			p.errorExpected("')'")
		}
		return &ConstDecl{Group: group, pos: constTok.Pos}
	}

	return p.parseConstSpec(constTok.Pos)
}

func (p *Parser) parseConstSpecInGroup(pos Position, lastType Expr, lastValues []Expr) *ConstDecl {
	if p.cur().Type != IDENT {
		p.errorExpected("identifier")
		return &ConstDecl{pos: pos}
	}

	name := &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos}
	p.advance()
	names := []*Ident{name}

	for p.cur().Type == COMMA {
		p.advance()
		if p.cur().Type != IDENT {
			p.errorExpected("identifier")
			break
		}
		names = append(names, &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos})
		p.advance()
	}

	var typeExpr Expr
	var valExprs []Expr

	if p.cur().Type == IDENT {
		typeExpr = &Ident{Name: p.cur().Lexeme, pos: p.cur().Pos}
		p.advance()
	}

	if p.cur().Type == ASSIGN {
		p.advance()
		val := p.parseExpr(0)
		valExprs = append(valExprs, val)
		for p.cur().Type == COMMA {
			p.advance()
			valExprs = append(valExprs, p.parseExpr(0))
		}
	} else if len(lastValues) > 0 {
		valExprs = lastValues
		if typeExpr == nil {
			typeExpr = lastType
		}
	} else {
		p.errorExpected("'='")
	}

	return &ConstDecl{
		Names:  names,
		Type:   typeExpr,
		Values: valExprs,
		pos:    pos,
	}
}

func (p *Parser) parseConstSpec(pos Position) *ConstDecl {
	return p.parseConstSpecInGroup(pos, nil, nil)
}

// FormatDiagnostics formats all recorded errors matching Slide 5c.
func (p *Parser) FormatDiagnostics() string {
	if len(p.errors) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, diag := range p.errors {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(diag.Format(p.srcLines))
	}
	sb.WriteString(fmt.Sprintf("\n%d errors, resynchronized each time\n", len(p.errors)))
	return sb.String()
}
