package main

import (
	"fmt"
	"reflect"
	"strings"
)

// Node represents any node in the AST.
type Node interface {
	Pos() Position
	formatTree(prefix string, isLast bool, isRoot bool) string
}

// Expr represents an expression node.
type Expr interface {
	Node
	exprNode()
}

// Stmt represents a statement node.
type Stmt interface {
	Node
	stmtNode()
}

// Decl represents a declaration node.
type Decl interface {
	Node
	declNode()
}

type childItem struct {
	label string
	node  Node
}

func isNilNode(node Node) bool {
	if node == nil {
		return true
	}
	v := reflect.ValueOf(node)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func formatTreeHelper(header string, children []childItem, prefix string, isLast bool, isRoot bool) string {
	var sb strings.Builder
	if isRoot {
		sb.WriteString(header)
	} else {
		branch := "├── "
		if isLast {
			branch = "└── "
		}
		sb.WriteString(prefix + branch + header)
	}

	nextPrefix := prefix
	if !isRoot {
		if isLast {
			nextPrefix += "    "
		} else {
			nextPrefix += "│   "
		}
	}

	var activeChildren []childItem
	for _, ch := range children {
		if !isNilNode(ch.node) {
			activeChildren = append(activeChildren, ch)
		}
	}

	for i, ch := range activeChildren {
		sb.WriteString("\n")
		lastChild := i == len(activeChildren)-1
		var childHeaderPrefix string
		if ch.label != "" {
			childHeaderPrefix = ch.label + ": "
		}
		sb.WriteString(formatChildWithLabel(childHeaderPrefix, ch.node, nextPrefix, lastChild))
	}
	return sb.String()
}

func formatChildWithLabel(label string, node Node, prefix string, isLast bool) string {
	if isNilNode(node) {
		return ""
	}
	raw := node.formatTree("", isLast, true)
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return ""
	}
	branch := "├── "
	if isLast {
		branch = "└── "
	}
	var sb strings.Builder
	sb.WriteString(prefix + branch + label + lines[0])

	nextPrefix := prefix
	if isLast {
		nextPrefix += "    "
	} else {
		nextPrefix += "│   "
	}

	for _, line := range lines[1:] {
		sb.WriteString("\n" + nextPrefix + line)
	}
	return sb.String()
}

// --- Expressions ---

type Ident struct {
	Name string
	pos  Position
}

func (i *Ident) Pos() Position                                   { return i.pos }
func (i *Ident) exprNode()                                       {}
func (i *Ident) formatTree(p string, isLast, isRoot bool) string {
	return formatTreeHelper(fmt.Sprintf("Ident %s", i.Name), nil, p, isLast, isRoot)
}

type BasicLit struct {
	Kind  TokenType
	Value string
	pos   Position
}

func (b *BasicLit) Pos() Position                           { return b.pos }
func (b *BasicLit) exprNode()                               {}
func (b *BasicLit) formatTree(p string, isLast, isRoot bool) string {
	name := "Lit"
	switch b.Kind {
	case INT:
		name = "Int"
	case FLOAT:
		name = "Float"
	case STRING:
		name = "String"
	}
	return formatTreeHelper(fmt.Sprintf("%s %s", name, b.Value), nil, p, isLast, isRoot)
}

type BinaryExpr struct {
	Op    TokenType
	OpStr string
	X     Expr
	Y     Expr
	pos   Position
}

func (b *BinaryExpr) Pos() Position { return b.pos }
func (b *BinaryExpr) exprNode()     {}
func (b *BinaryExpr) formatTree(p string, isLast, isRoot bool) string {
	header := fmt.Sprintf("Binary %s", b.OpStr)
	children := []childItem{
		{label: "X", node: b.X},
		{label: "Y", node: b.Y},
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type UnaryExpr struct {
	Op    TokenType
	OpStr string
	X     Expr
	pos   Position
}

func (u *UnaryExpr) Pos() Position { return u.pos }
func (u *UnaryExpr) exprNode()     {}
func (u *UnaryExpr) formatTree(p string, isLast, isRoot bool) string {
	header := fmt.Sprintf("Unary %s", u.OpStr)
	children := []childItem{
		{label: "X", node: u.X},
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type ParenExpr struct {
	X   Expr
	pos Position
}

func (pe *ParenExpr) Pos() Position { return pe.pos }
func (pe *ParenExpr) exprNode()     {}
func (pe *ParenExpr) formatTree(p string, isLast, isRoot bool) string {
	header := "Paren"
	children := []childItem{
		{label: "", node: pe.X},
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type CallExpr struct {
	Fun  Expr
	Args []Expr
	pos  Position
}

func (c *CallExpr) Pos() Position { return c.pos }
func (c *CallExpr) exprNode()     {}
func (c *CallExpr) formatTree(p string, isLast, isRoot bool) string {
	header := "CallExpr"
	children := []childItem{
		{label: "Fun", node: c.Fun},
	}
	for i, arg := range c.Args {
		children = append(children, childItem{
			label: fmt.Sprintf("Arg[%d]", i),
			node:  arg,
		})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type IndexExpr struct {
	X     Expr
	Index Expr
	pos   Position
}

func (ie *IndexExpr) Pos() Position { return ie.pos }
func (ie *IndexExpr) exprNode()     {}
func (ie *IndexExpr) formatTree(p string, isLast, isRoot bool) string {
	header := "IndexExpr"
	children := []childItem{
		{label: "X", node: ie.X},
		{label: "Index", node: ie.Index},
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type SelectorExpr struct {
	X   Expr
	Sel *Ident
	pos Position
}

func (s *SelectorExpr) Pos() Position { return s.pos }
func (s *SelectorExpr) exprNode()     {}
func (s *SelectorExpr) formatTree(p string, isLast, isRoot bool) string {
	if s == nil {
		return "<nil>"
	}
	header := "SelectorExpr"
	children := []childItem{
		{label: "X", node: s.X},
		{label: "Sel", node: s.Sel},
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type BadExpr struct {
	pos Position
}

func (b *BadExpr) Pos() Position                           { return b.pos }
func (b *BadExpr) exprNode()                               {}
func (b *BadExpr) formatTree(p string, isLast, isRoot bool) string {
	return formatTreeHelper(fmt.Sprintf("BadExpr (%s)", b.pos), nil, p, isLast, isRoot)
}

// --- Statements ---

type AssignStmt struct {
	Tok     TokenType
	OpStr   string
	Lhs     Expr
	Rhs     Expr
	LhsList []Expr
	RhsList []Expr
	pos     Position
}

func (a *AssignStmt) Pos() Position { return a.pos }
func (a *AssignStmt) stmtNode()     {}
func (a *AssignStmt) formatTree(p string, isLast, isRoot bool) string {
	header := fmt.Sprintf("AssignStmt (%s)", a.pos)
	children := []childItem{
		{label: "Lhs", node: a.Lhs},
		{label: "Rhs", node: a.Rhs},
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type IncDecStmt struct {
	X     Expr
	Tok   TokenType
	OpStr string
	pos   Position
}

func (ids *IncDecStmt) Pos() Position { return ids.pos }
func (ids *IncDecStmt) stmtNode()     {}
func (ids *IncDecStmt) formatTree(p string, isLast, isRoot bool) string {
	if ids == nil {
		return "<nil>"
	}
	header := fmt.Sprintf("IncDecStmt %s (%s)", ids.OpStr, ids.pos)
	children := []childItem{
		{label: "X", node: ids.X},
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type ExprStmt struct {
	X   Expr
	pos Position
}

func (e *ExprStmt) Pos() Position { return e.pos }
func (e *ExprStmt) stmtNode()     {}
func (e *ExprStmt) formatTree(p string, isLast, isRoot bool) string {
	header := fmt.Sprintf("ExprStmt (%s)", e.pos)
	children := []childItem{
		{label: "X", node: e.X},
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type BlockStmt struct {
	List []Stmt
	pos  Position
}

func (b *BlockStmt) Pos() Position { return b.pos }
func (b *BlockStmt) stmtNode()     {}
func (b *BlockStmt) formatTree(p string, isLast, isRoot bool) string {
	header := fmt.Sprintf("BlockStmt (%s)", b.pos)
	var children []childItem
	for i, s := range b.List {
		children = append(children, childItem{
			label: fmt.Sprintf("[%d]", i),
			node:  s,
		})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type IfStmt struct {
	Init Stmt
	Cond Expr
	Body *BlockStmt
	Else Stmt
	pos  Position
}

func (i *IfStmt) Pos() Position { return i.pos }
func (i *IfStmt) stmtNode()     {}
func (i *IfStmt) formatTree(p string, isLast, isRoot bool) string {
	header := fmt.Sprintf("IfStmt (%s)", i.pos)
	var children []childItem
	if i.Init != nil {
		children = append(children, childItem{label: "Init", node: i.Init})
	}
	children = append(children, childItem{label: "Cond", node: i.Cond})
	children = append(children, childItem{label: "Body", node: i.Body})
	if i.Else != nil {
		children = append(children, childItem{label: "Else", node: i.Else})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type ForStmt struct {
	Init Stmt
	Cond Expr
	Post Stmt
	Body *BlockStmt
	pos  Position
}

func (f *ForStmt) Pos() Position { return f.pos }
func (f *ForStmt) stmtNode()     {}
func (f *ForStmt) formatTree(p string, isLast, isRoot bool) string {
	header := fmt.Sprintf("ForStmt (%s)", f.pos)
	var children []childItem
	if f.Init != nil {
		children = append(children, childItem{label: "Init", node: f.Init})
	}
	if f.Cond != nil {
		children = append(children, childItem{label: "Cond", node: f.Cond})
	}
	if f.Post != nil {
		children = append(children, childItem{label: "Post", node: f.Post})
	}
	children = append(children, childItem{label: "Body", node: f.Body})
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type ReturnStmt struct {
	Results []Expr
	pos     Position
}

func (r *ReturnStmt) Pos() Position { return r.pos }
func (r *ReturnStmt) stmtNode()     {}
func (r *ReturnStmt) formatTree(p string, isLast, isRoot bool) string {
	header := fmt.Sprintf("ReturnStmt (%s)", r.pos)
	var children []childItem
	for i, res := range r.Results {
		children = append(children, childItem{
			label: fmt.Sprintf("Result[%d]", i),
			node:  res,
		})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type BranchStmt struct {
	Tok TokenType
	pos Position
}

func (b *BranchStmt) Pos() Position                           { return b.pos }
func (b *BranchStmt) stmtNode()                               {}
func (b *BranchStmt) formatTree(p string, isLast, isRoot bool) string {
	return formatTreeHelper(fmt.Sprintf("BranchStmt %s (%s)", b.Tok, b.pos), nil, p, isLast, isRoot)
}

type BadStmt struct {
	pos Position
}

func (b *BadStmt) Pos() Position                           { return b.pos }
func (b *BadStmt) stmtNode()                               {}
func (b *BadStmt) formatTree(p string, isLast, isRoot bool) string {
	return formatTreeHelper(fmt.Sprintf("BadStmt (%s)", b.pos), nil, p, isLast, isRoot)
}

// --- Declarations ---

type ImportSpec struct {
	Name *Ident
	Path *BasicLit
	pos  Position
}

func (s *ImportSpec) Pos() Position { return s.pos }
func (s *ImportSpec) formatTree(p string, isLast, isRoot bool) string {
	if s == nil {
		return "<nil>"
	}
	pathVal := `""`
	if s.Path != nil {
		pathVal = s.Path.Value
	}
	header := fmt.Sprintf("ImportSpec %s (%s)", pathVal, s.pos)
	if s.Name != nil {
		header = fmt.Sprintf("ImportSpec %s %s (%s)", s.Name.Name, pathVal, s.pos)
	}
	return formatTreeHelper(header, nil, p, isLast, isRoot)
}

type ImportDecl struct {
	Specs []*ImportSpec
	pos   Position
}

func (i *ImportDecl) Pos() Position { return i.pos }
func (i *ImportDecl) declNode()     {}
func (i *ImportDecl) stmtNode()     {}
func (i *ImportDecl) formatTree(p string, isLast, isRoot bool) string {
	if i == nil {
		return "<nil>"
	}
	header := fmt.Sprintf("ImportDecl (%s)", i.pos)
	var children []childItem
	for _, spec := range i.Specs {
		children = append(children, childItem{label: "", node: spec})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type VarDecl struct {
	Name   *Ident
	Names  []*Ident
	Type   Expr
	Value  Expr
	Values []Expr
	Group  []*VarDecl
	pos    Position
}

func (v *VarDecl) Pos() Position { return v.pos }
func (v *VarDecl) declNode()     {}
func (v *VarDecl) stmtNode()     {}
func (v *VarDecl) formatTree(p string, isLast, isRoot bool) string {
	if v == nil {
		return "<nil>"
	}
	name := "<unnamed>"
	if len(v.Names) > 0 {
		var nStrs []string
		for _, n := range v.Names {
			nStrs = append(nStrs, n.Name)
		}
		name = strings.Join(nStrs, ", ")
	} else if v.Name != nil {
		name = v.Name.Name
	}
	header := fmt.Sprintf("VarDecl %s (%s)", name, v.pos)
	var children []childItem
	if v.Type != nil {
		children = append(children, childItem{label: "Type", node: v.Type})
	}
	if v.Value != nil {
		children = append(children, childItem{label: "Value", node: v.Value})
	}
	for i, val := range v.Values {
		children = append(children, childItem{label: fmt.Sprintf("Value[%d]", i), node: val})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type Field struct {
	Names []*Ident
	Type  Expr
}

type FuncDecl struct {
	Recv    *Field // Method receiver, e.g. (c Counter)
	Name    *Ident
	Params  []*Field
	Type    Expr
	Results []*Field // Multiple return values
	Body    *BlockStmt
	pos     Position
}

func (f *FuncDecl) Pos() Position { return f.pos }
func (f *FuncDecl) declNode()     {}
func (f *FuncDecl) formatTree(p string, isLast, isRoot bool) string {
	if f == nil {
		return "<nil>"
	}
	name := "<unnamed>"
	if f.Name != nil {
		name = f.Name.Name
	}
	header := fmt.Sprintf("FuncDecl %s (%s)", name, f.pos)
	var children []childItem
	if len(f.Params) > 0 {
		var paramDescs []string
		for _, param := range f.Params {
			if param == nil {
				continue
			}
			var names []string
			for _, n := range param.Names {
				if n != nil {
					names = append(names, n.Name)
				}
			}
			typeStr := "unknown"
			if ident, ok := param.Type.(*Ident); ok && ident != nil {
				typeStr = ident.Name
			}
			paramDescs = append(paramDescs, fmt.Sprintf("%s %s", strings.Join(names, ", "), typeStr))
		}
		header += fmt.Sprintf(" (%s)", strings.Join(paramDescs, ", "))
	}
	if f.Type != nil {
		children = append(children, childItem{label: "Type", node: f.Type})
	}
	if f.Body != nil {
		children = append(children, childItem{label: "Body", node: f.Body})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type BadDecl struct {
	pos Position
}

func (b *BadDecl) Pos() Position                           { return b.pos }
func (b *BadDecl) declNode()                               {}
func (b *BadDecl) formatTree(p string, isLast, isRoot bool) string {
	return formatTreeHelper(fmt.Sprintf("BadDecl (%s)", b.pos), nil, p, isLast, isRoot)
}

// --- Extended Phase 1 AST Nodes ---

type TypeDecl struct {
	Name *Ident
	Type Expr
	pos  Position
}

func (t *TypeDecl) Pos() Position { return t.pos }
func (t *TypeDecl) declNode()     {}
func (t *TypeDecl) stmtNode()     {}
func (t *TypeDecl) formatTree(p string, isLast, isRoot bool) string {
	if t == nil {
		return "<nil>"
	}
	name := "<unnamed>"
	if t.Name != nil {
		name = t.Name.Name
	}
	header := fmt.Sprintf("TypeDecl %s (%s)", name, t.pos)
	children := []childItem{
		{label: "Type", node: t.Type},
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type StructType struct {
	Fields []*Field
	pos    Position
}

func (st *StructType) Pos() Position { return st.pos }
func (st *StructType) exprNode()     {}
func (st *StructType) formatTree(p string, isLast, isRoot bool) string {
	if st == nil {
		return "<nil>"
	}
	header := fmt.Sprintf("StructType (%s)", st.pos)
	var children []childItem
	for i, f := range st.Fields {
		var nStrs []string
		for _, n := range f.Names {
			nStrs = append(nStrs, n.Name)
		}
		label := fmt.Sprintf("Field[%d] %s", i, strings.Join(nStrs, ", "))
		children = append(children, childItem{label: label, node: f.Type})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type ArrayType struct {
	Len Expr // nil for slice []T
	Elt Expr // element type T
	pos Position
}

func (at *ArrayType) Pos() Position { return at.pos }
func (at *ArrayType) exprNode()     {}
func (at *ArrayType) formatTree(p string, isLast, isRoot bool) string {
	if at == nil {
		return "<nil>"
	}
	header := "SliceType"
	if at.Len != nil {
		header = "ArrayType"
	}
	header += fmt.Sprintf(" (%s)", at.pos)
	var children []childItem
	if at.Len != nil {
		children = append(children, childItem{label: "Len", node: at.Len})
	}
	children = append(children, childItem{label: "Elt", node: at.Elt})
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type CompositeLit struct {
	Type Expr
	Elts []Expr
	pos  Position
}

func (cl *CompositeLit) Pos() Position { return cl.pos }
func (cl *CompositeLit) exprNode()     {}
func (cl *CompositeLit) formatTree(p string, isLast, isRoot bool) string {
	if cl == nil {
		return "<nil>"
	}
	header := fmt.Sprintf("CompositeLit (%s)", cl.pos)
	var children []childItem
	if cl.Type != nil {
		children = append(children, childItem{label: "Type", node: cl.Type})
	}
	for i, elt := range cl.Elts {
		children = append(children, childItem{label: fmt.Sprintf("[%d]", i), node: elt})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type KeyValueExpr struct {
	Key   Expr
	Value Expr
	pos   Position
}

func (kv *KeyValueExpr) Pos() Position { return kv.pos }
func (kv *KeyValueExpr) exprNode()     {}
func (kv *KeyValueExpr) formatTree(p string, isLast, isRoot bool) string {
	if kv == nil {
		return "<nil>"
	}
	header := fmt.Sprintf("KeyValueExpr (%s)", kv.pos)
	children := []childItem{
		{label: "Key", node: kv.Key},
		{label: "Value", node: kv.Value},
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type SliceExpr struct {
	X    Expr
	Low  Expr
	High Expr
	Max  Expr
	pos  Position
}

func (se *SliceExpr) Pos() Position { return se.pos }
func (se *SliceExpr) exprNode()     {}
func (se *SliceExpr) formatTree(p string, isLast, isRoot bool) string {
	if se == nil {
		return "<nil>"
	}
	header := fmt.Sprintf("SliceExpr (%s)", se.pos)
	var children []childItem
	children = append(children, childItem{label: "X", node: se.X})
	if se.Low != nil {
		children = append(children, childItem{label: "Low", node: se.Low})
	}
	if se.High != nil {
		children = append(children, childItem{label: "High", node: se.High})
	}
	if se.Max != nil {
		children = append(children, childItem{label: "Max", node: se.Max})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type CaseClause struct {
	List []Expr // nil means default:
	Body []Stmt
	pos  Position
}

func (cc *CaseClause) Pos() Position { return cc.pos }
func (cc *CaseClause) stmtNode()     {}
func (cc *CaseClause) formatTree(p string, isLast, isRoot bool) string {
	if cc == nil {
		return "<nil>"
	}
	header := "CaseClause"
	if len(cc.List) == 0 {
		header = "DefaultClause"
	}
	header += fmt.Sprintf(" (%s)", cc.pos)
	var children []childItem
	for i, expr := range cc.List {
		children = append(children, childItem{label: fmt.Sprintf("Cond[%d]", i), node: expr})
	}
	for i, stmt := range cc.Body {
		children = append(children, childItem{label: fmt.Sprintf("Stmt[%d]", i), node: stmt})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type SwitchStmt struct {
	Init Stmt
	Tag  Expr
	Body []*CaseClause
	pos  Position
}

func (sw *SwitchStmt) Pos() Position { return sw.pos }
func (sw *SwitchStmt) stmtNode()     {}
func (sw *SwitchStmt) formatTree(p string, isLast, isRoot bool) string {
	if sw == nil {
		return "<nil>"
	}
	header := fmt.Sprintf("SwitchStmt (%s)", sw.pos)
	var children []childItem
	if sw.Init != nil {
		children = append(children, childItem{label: "Init", node: sw.Init})
	}
	if sw.Tag != nil {
		children = append(children, childItem{label: "Tag", node: sw.Tag})
	}
	for i, clause := range sw.Body {
		children = append(children, childItem{label: fmt.Sprintf("Clause[%d]", i), node: clause})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type RangeStmt struct {
	Key   Expr
	Value Expr
	Tok   TokenType // DEFINE or ASSIGN
	X     Expr
	Body  *BlockStmt
	pos   Position
}

func (rs *RangeStmt) Pos() Position { return rs.pos }
func (rs *RangeStmt) stmtNode()     {}
func (rs *RangeStmt) formatTree(p string, isLast, isRoot bool) string {
	if rs == nil {
		return "<nil>"
	}
	header := fmt.Sprintf("ForRangeStmt (%s)", rs.pos)
	var children []childItem
	if rs.Key != nil {
		children = append(children, childItem{label: "Key", node: rs.Key})
	}
	if rs.Value != nil {
		children = append(children, childItem{label: "Value", node: rs.Value})
	}
	children = append(children, childItem{label: "X", node: rs.X})
	children = append(children, childItem{label: "Body", node: rs.Body})
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type DeferStmt struct {
	Call *CallExpr
	pos  Position
}

func (ds *DeferStmt) Pos() Position { return ds.pos }
func (ds *DeferStmt) stmtNode()     {}
func (ds *DeferStmt) formatTree(p string, isLast, isRoot bool) string {
	if ds == nil {
		return "<nil>"
	}
	header := fmt.Sprintf("DeferStmt (%s)", ds.pos)
	children := []childItem{
		{label: "Call", node: ds.Call},
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type ConstDecl struct {
	Names  []*Ident
	Type   Expr
	Values []Expr
	Group  []*ConstDecl
	pos    Position
}

func (cd *ConstDecl) Pos() Position { return cd.pos }
func (cd *ConstDecl) declNode()     {}
func (cd *ConstDecl) stmtNode()     {}
func (cd *ConstDecl) formatTree(p string, isLast, isRoot bool) string {
	if cd == nil {
		return "<nil>"
	}
	name := "<unnamed>"
	if len(cd.Names) > 0 {
		var nStrs []string
		for _, n := range cd.Names {
			nStrs = append(nStrs, n.Name)
		}
		name = strings.Join(nStrs, ", ")
	}
	header := fmt.Sprintf("ConstDecl %s (%s)", name, cd.pos)
	var children []childItem
	if cd.Type != nil {
		children = append(children, childItem{label: "Type", node: cd.Type})
	}
	for i, val := range cd.Values {
		children = append(children, childItem{label: fmt.Sprintf("Value[%d]", i), node: val})
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

type MapType struct {
	Key   Expr
	Value Expr
	pos   Position
}

func (m *MapType) Pos() Position { return m.pos }
func (m *MapType) exprNode()     {}
func (m *MapType) formatTree(p string, isLast, isRoot bool) string {
	if m == nil {
		return "<nil>"
	}
	header := fmt.Sprintf("MapType (%s)", m.pos)
	children := []childItem{
		{label: "Key", node: m.Key},
		{label: "Value", node: m.Value},
	}
	return formatTreeHelper(header, children, p, isLast, isRoot)
}

// --- File ---

type File struct {
	Name       *Ident
	Decls      []Decl
	Stmts      []Stmt
	PackagePos Position
	pos        Position
}

func (f *File) Pos() Position { return f.pos }
func (f *File) formatTree(p string, isLast, isRoot bool) string {
	name := "File"
	if f.Name != nil {
		name = fmt.Sprintf("File %s (%s)", f.Name.Name, f.pos)
	}
	var children []childItem
	for _, d := range f.Decls {
		children = append(children, childItem{label: "", node: d})
	}
	for _, s := range f.Stmts {
		children = append(children, childItem{label: "", node: s})
	}
	return formatTreeHelper(name, children, p, isLast, isRoot)
}

// PrintAST returns the formatted ASCII tree for any Node.
func PrintAST(node Node) string {
	if isNilNode(node) {
		return "<nil>"
	}
	return node.formatTree("", true, true)
}
