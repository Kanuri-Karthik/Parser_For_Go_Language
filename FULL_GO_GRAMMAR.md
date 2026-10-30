# Complete Formal Grammar for the Go Programming Language

**Standard:** The Go Programming Language Specification (Official Specification, Go 1.18+ including Generics)  
**Notation:** Extended Backus-Naur Form (ISO/IEC 14977 & Go Spec EBNF)  
**File Location:** `d:\Downloads\CD_CBP\FULL_GO_GRAMMAR.md`

---

## Table of Contents

1. [Notation Conventions](#1-notation-conventions)
2. [Lexical Grammar](#2-lexical-grammar)
   - [Characters, Whitespace & Comments](#characters-whitespace--comments)
   - [Identifiers & Keywords](#identifiers--keywords)
   - [Operators & Delimiters](#operators--delimiters)
   - [Integer, Floating-point, Imaginary & Rune Literals](#literals)
   - [String Literals](#string-literals)
   - [Automatic Semicolon Insertion (ASI)](#automatic-semicolon-insertion-asi)
3. [Type Grammar](#3-type-grammar)
   - [Type Names & Generics](#type-names--type-parameters)
   - [Composite Types (Array, Slice, Struct, Pointer)](#composite-types)
   - [Function, Interface, Map & Channel Types](#function-interface-map--channel-types)
4. [Declarations & Scope](#4-declarations--scope)
   - [Constant Declarations](#constant-declarations)
   - [Type Declarations & Aliases](#type-declarations)
   - [Variable Declarations](#variable-declarations)
   - [Function & Method Declarations](#function--method-declarations)
5. [Expressions & Operators](#5-expressions--operators)
   - [Operands & Composite Literals](#operands--composite-literals)
   - [Primary Expressions (Calls, Index, Slice, Selectors, Type Assertions)](#primary-expressions)
   - [Unary & Binary Operator Precedence](#operators--precedence)
   - [Conversions](#conversions)
6. [Statements Grammar](#6-statements-grammar)
   - [Simple Statements (Assignment, Short Var, Inc/Dec, Send)](#simple-statements)
   - [Control Flow (If, For, Range, Switch, Type-Switch, Select)](#control-flow-statements)
   - [Jump & Concurrency Statements (Go, Defer, Return, Break, Continue, Goto, Fallthrough)](#jump--concurrency-statements)
7. [Source File & Package Grammar](#7-source-file--package-grammar)
8. [Comparison: Full Go vs. Project Subset](#8-comparison-full-go-vs-project-subset)

---

## 1. Notation Conventions

The grammar is specified using Extended Backus-Naur Form (EBNF):

```
|   alternation
()  grouping
[]  optional (0 or 1 times)
{}  repetition (0 to n times)
```

Lower-case production names identify lexical tokens. Non-terminals are in `CamelCase`. Lexical terminals are enclosed in double quotes `""` or backticks ````.

---

## 2. Lexical Grammar

### Characters, Whitespace & Comments

```ebnf
newline        = /* the Unicode code point U+000A (LF) */ ;
unicode_char   = /* an arbitrary Unicode code point except newline */ ;
unicode_letter = /* a Unicode code point categorized as "Letter" */ ;
unicode_digit  = /* a Unicode code point categorized as "Number, decimal digit" */ ;

letter         = unicode_letter | "_" ;
decimal_digit  = "0" … "9" ;
binary_digit   = "0" | "1" ;
octal_digit    = "0" … "7" ;
hex_digit      = "0" … "9" | "A" … "F" | "a" … "f" ;

comment        = line_comment | general_comment ;
line_comment   = "//" { unicode_char } newline ;
general_comment= "/*" { unicode_char | newline } "*/" ;
```

### Identifiers & Keywords

```ebnf
identifier     = letter { letter | unicode_digit } ;
```

**The 25 Reserved Keywords:**
```
break        default      func         interface    select
case         defer        go           map          struct
chan         else         goto         package      switch
const        fallthrough  if           range        type
continue     for          import       return       var
```

### Operators & Delimiters

```
+    &     +=    &=     &&    ==    !=    (    )
-    |     -=    |=     ||    <     <=    [    ]
*    ^     *=    ^=     <-    >     >=    {    }
/    <<    /=    <<=    ++    =     :=    ,    ;
%    >>    %=    >>=    --    !     ...   .    :
     &^          &^=          ~
```

### Literals

```ebnf
int_lit        = decimal_lit | binary_lit | octal_lit | hex_lit ;
decimal_lit    = "0" | ( "1" … "9" ) [ [ "_" ] decimal_digits ] ;
binary_lit     = "0" ( "b" | "B" ) [ "_" ] binary_digits ;
octal_lit      = "0" [ "o" | "O" ] [ "_" ] octal_digits ;
hex_lit        = "0" ( "x" | "X" ) [ "_" ] hex_digits ;

decimal_digits = decimal_digit { [ "_" ] decimal_digit } ;
binary_digits  = binary_digit { [ "_" ] binary_digit } ;
octal_digits   = octal_digit { [ "_" ] octal_digit } ;
hex_digits     = hex_digit { [ "_" ] hex_digit } ;

float_lit      = decimal_float_lit | hex_float_lit ;
decimal_float_lit = decimal_digits "." [ decimal_digits ] [ decimal_exponent ]
                  | decimal_digits decimal_exponent
                  | "." decimal_digits [ decimal_exponent ] ;
decimal_exponent  = ( "e" | "E" ) [ "+" | "-" ] decimal_digits ;

hex_float_lit  = "0" ( "x" | "X" ) hex_mantissa hex_exponent ;
hex_mantissa   = [ "_" ] hex_digits "." [ hex_digits ]
               | [ "_" ] hex_digits
               | "." hex_digits ;
hex_exponent   = ( "p" | "P" ) [ "+" | "-" ] decimal_digits ;

imaginary_lit  = ( decimal_digits | int_lit | float_lit ) "i" ;

rune_lit       = "'" ( unicode_value | byte_value ) "'" ;
unicode_value  = unicode_char | little_u_value | big_u_value | escaped_char ;
byte_value     = octal_byte_value | hex_byte_value ;
octal_byte_value = `\` octal_digit octal_digit octal_digit ;
hex_byte_value   = `\` "x" hex_digit hex_digit ;
little_u_value   = `\` "u" hex_digit hex_digit hex_digit hex_digit ;
big_u_value      = `\` "U" hex_digit hex_digit hex_digit hex_digit hex_digit hex_digit hex_digit hex_digit ;
escaped_char     = `\` ( "a" | "b" | "f" | "n" | "r" | "t" | "v" | `\` | "'" | `"` ) ;
```

### String Literals

```ebnf
string_lit             = raw_string_lit | interpreted_string_lit ;
raw_string_lit         = "`" { unicode_char | newline } "`" ;
interpreted_string_lit = `"` { unicode_value | byte_value } `"` ;
```

### Automatic Semicolon Insertion (ASI)

When the input is broken into tokens, a semicolon is automatically inserted into the token stream immediately after a line's final token if that token is:
1. An `identifier`
2. An integer, float, imaginary, rune, or string literal (`int_lit`, `float_lit`, `imaginary_lit`, `rune_lit`, `string_lit`)
3. One of the jump keywords: `break`, `continue`, `fallthrough`, or `return`
4. One of the operators or delimiters: `++`, `--`, `)`, `]`, or `}`

---

## 3. Type Grammar

```ebnf
Type           = TypeName [ TypeArgs ] | TypeLit | "(" Type ")" ;
TypeName       = identifier | QualifiedIdent ;
TypeArgs       = "[" TypeList [ "," ] "]" ;
TypeList       = Type { "," Type } ;

TypeLit        = ArrayType
               | StructType
               | PointerType
               | FunctionType
               | InterfaceType
               | SliceType
               | MapType
               | ChannelType ;
```

### Composite Types

```ebnf
(* Arrays and Slices *)
ArrayType      = "[" ArrayLength "]" ElementType ;
ArrayLength    = Expression ;
ElementType    = Type ;
SliceType      = "[" "]" ElementType ;

(* Structs *)
StructType     = "struct" "{" { FieldDecl ";" } "}" ;
FieldDecl      = ( IdentifierList Type | EmbeddedField ) [ Tag ] ;
EmbeddedField  = [ "*" ] TypeName [ TypeArgs ] ;
Tag            = string_lit ;

(* Pointers *)
PointerType    = "*" BaseType ;
BaseType       = Type ;
```

### Function, Interface, Map & Channel Types

```ebnf
(* Function Types *)
FunctionType   = "func" Signature ;
Signature      = Parameters [ Result ] ;
Result         = Parameters | Type ;
Parameters     = "(" [ ParameterList [ "," ] ] ")" ;
ParameterList  = ParameterDecl { "," ParameterDecl } ;
ParameterDecl  = [ IdentifierList ] [ "..." ] Type ;

(* Interface Types & Type Constraints (Generics) *)
InterfaceType  = "interface" "{" { InterfaceElem ";" } "}" ;
InterfaceElem  = MethodElem | TypeElem ;
MethodElem     = MethodName Signature ;
MethodName     = identifier ;
TypeElem       = TypeTerm { "|" TypeTerm } ;
TypeTerm       = Type | "~" Type ;

(* Map Types *)
MapType        = "map" "[" KeyType "]" ElementType ;
KeyType        = Type ;

(* Channel Types *)
ChannelType    = ( "chan" | "chan" "<-" | "<-" "chan" ) ElementType ;
```

---

## 4. Declarations & Scope

```ebnf
Declaration    = ConstDecl | TypeDecl | VarDecl ;
TopLevelDecl   = Declaration | FunctionDecl | MethodDecl ;
```

### Constant Declarations

```ebnf
ConstDecl      = "const" ( ConstSpec | "(" { ConstSpec ";" } ")" ) ;
ConstSpec      = IdentifierList [ [ Type ] "=" ExpressionList ] ;
IdentifierList = identifier { "," identifier } ;
ExpressionList = Expression { "," Expression } ;
```

### Type Declarations & Generics

```ebnf
TypeDecl       = "type" ( TypeSpec | "(" { TypeSpec ";" } ")" ) ;
TypeSpec       = AliasDecl | TypeDef ;
AliasDecl      = identifier [ TypeParameters ] "=" Type ;
TypeDef        = identifier [ TypeParameters ] Type ;

TypeParameters = "[" TypeParamList [ "," ] "]" ;
TypeParamList  = TypeParamDecl { "," TypeParamDecl } ;
TypeParamDecl  = IdentifierList TypeConstraint ;
TypeConstraint = TypeElem ;
```

### Variable Declarations

```ebnf
VarDecl        = "var" ( VarSpec | "(" { VarSpec ";" } ")" ) ;
VarSpec        = IdentifierList ( Type [ "=" ExpressionList ] | "=" ExpressionList ) ;
```

### Function & Method Declarations

```ebnf
FunctionDecl   = "func" FunctionName [ TypeParameters ] Signature [ FunctionBody ] ;
FunctionName   = identifier ;
FunctionBody   = Block ;

MethodDecl     = "func" Receiver MethodName Signature [ FunctionBody ] ;
Receiver       = Parameters ;
```

---

## 5. Expressions & Operators

### Operands & Composite Literals

```ebnf
Operand        = Literal | OperandName [ TypeArgs ] | "(" Expression ")" ;
Literal        = BasicLit | CompositeLit | FunctionLit ;
BasicLit       = int_lit | float_lit | imaginary_lit | rune_lit | string_lit ;
OperandName    = identifier | QualifiedIdent ;
QualifiedIdent = PackageName "." identifier ;

CompositeLit   = LiteralType LiteralValue ;
LiteralType    = StructType | ArrayType | "[" "..." "]" ElementType
               | SliceType | MapType | TypeName [ TypeArgs ] ;
LiteralValue   = "{" [ ElementList [ "," ] ] "}" ;
ElementList    = KeyedElement { "," KeyedElement } ;
KeyedElement   = [ Key ":" ] Element ;
Key            = FieldName | Expression | LiteralValue ;
FieldName      = identifier ;
Element        = Expression | LiteralValue ;

FunctionLit    = "func" Signature FunctionBody ;
```

### Primary Expressions

```ebnf
PrimaryExpr    = Operand
               | Conversion
               | MethodExpr
               | PrimaryExpr Selector
               | PrimaryExpr Index
               | PrimaryExpr Slice
               | PrimaryExpr TypeAssertion
               | PrimaryExpr Arguments ;

Selector       = "." identifier ;
Index          = "[" Expression [ "," ] "]" ;
Slice          = "[" [ Expression ] ":" [ Expression ] "]"
               | "[" [ Expression ] ":" Expression ":" Expression "]" ;
TypeAssertion  = "." "(" Type ")" ;
Arguments      = "(" [ ( ExpressionList | Type [ "," ExpressionList ] ) [ "..." ] [ "," ] ] ")" ;
MethodExpr     = ReceiverType "." MethodName ;
ReceiverType   = Type ;
```

### Operators & Precedence

Binary operators are divided into 5 strictly-ordered precedence levels:

```ebnf
Expression     = UnaryExpr | Expression binary_op Expression ;
UnaryExpr      = PrimaryExpr | unary_op UnaryExpr ;

binary_op      = "||"
               | "&&"
               | rel_op
               | add_op
               | mul_op ;

rel_op         = "==" | "!=" | "<" | "<=" | ">" | ">=" ;
add_op         = "+" | "-" | "|" | "^" ;
mul_op         = "*" | "/" | "%" | "<<" | ">>" | "&" | "&^" ;

unary_op       = "+" | "-" | "!" | "^" | "*" | "&" | "<-" ;
```

#### Precedence Hierarchy (Highest to Lowest)
1. **Level 5 (Highest binary)**: `*`, `/`, `%`, `<<`, `>>`, `&`, `&^`
2. **Level 4**: `+`, `-`, `|`, `^`
3. **Level 3**: `==`, `!=`, `<`, `<=`, `>`, `>=`
4. **Level 2**: `&&`
5. **Level 1 (Lowest binary)**: `||`

*Note: Unary operators (`+`, `-`, `!`, `^`, `*`, `&`, `<-`) have higher precedence than any binary operator.*

### Conversions

```ebnf
Conversion     = Type "(" Expression [ "," ] ")" ;
```

---

## 6. Statements Grammar

```ebnf
Statement      = Declaration
               | LabeledStmt
               | SimpleStmt
               | GoStmt
               | ReturnStmt
               | BreakStmt
               | ContinueStmt
               | GotoStmt
               | FallthroughStmt
               | Block
               | IfStmt
               | SwitchStmt
               | SelectStmt
               | ForStmt
               | DeferStmt ;

Block          = "{" StatementList "}" ;
StatementList  = { Statement ";" } ;
```

### Simple Statements

```ebnf
SimpleStmt     = EmptyStmt
               | ExpressionStmt
               | SendStmt
               | IncDecStmt
               | Assignment
               | ShortVarDecl ;

EmptyStmt      = ;
ExpressionStmt = Expression ;
SendStmt       = Channel "<-" Expression ;
Channel        = Expression ;
IncDecStmt     = Expression ( "++" | "--" ) ;
Assignment     = ExpressionList assign_op ExpressionList ;
assign_op      = [ add_op | mul_op ] "=" ;
ShortVarDecl   = IdentifierList ":=" ExpressionList ;
```

### Control Flow Statements

#### If Statements
```ebnf
IfStmt         = "if" [ SimpleStmt ";" ] Expression Block [ "else" ( IfStmt | Block ) ] ;
```

#### Switch Statements (Expression & Type Switches)
```ebnf
SwitchStmt     = ExprSwitchStmt | TypeSwitchStmt ;

ExprSwitchStmt = "switch" [ SimpleStmt ";" ] [ Expression ] "{" { ExprCaseClause } "}" ;
ExprCaseClause = ExprSwitchCase ":" StatementList ;
ExprSwitchCase = "case" ExpressionList | "default" ;

TypeSwitchStmt = "switch" [ SimpleStmt ";" ] TypeSwitchGuard "{" { TypeCaseClause } "}" ;
TypeSwitchGuard= [ identifier ":=" ] PrimaryExpr "." "(" "type" ")" ;
TypeCaseClause = TypeSwitchCase ":" StatementList ;
TypeSwitchCase = "case" TypeList | "default" ;
```

#### For Statements (Loop, Condition & Range)
```ebnf
ForStmt        = "for" [ Condition | ForClause | RangeClause ] Block ;
Condition      = Expression ;
ForClause      = [ InitStmt ] ";" [ Condition ] ";" [ PostStmt ] ;
InitStmt       = SimpleStmt ;
PostStmt       = SimpleStmt ;

RangeClause    = [ ExpressionList "=" | IdentifierList ":=" ] "range" Expression ;
```

#### Select Statements (Channel Multiplexing)
```ebnf
SelectStmt     = "select" "{" { CommClause } "}" ;
CommClause     = CommCase ":" StatementList ;
CommCase       = "case" ( SendStmt | RecvStmt ) | "default" ;
RecvStmt       = [ ExpressionList "=" | IdentifierList ":=" ] RecvExpr ;
RecvExpr       = Expression ;
```

### Jump & Concurrency Statements

```ebnf
LabeledStmt      = Label ":" Statement ;
Label            = identifier ;

GoStmt           = "go" Expression ;
DeferStmt        = "defer" Expression ;
ReturnStmt       = "return" [ ExpressionList ] ;
BreakStmt        = "break" [ Label ] ;
ContinueStmt     = "continue" [ Label ] ;
GotoStmt         = "goto" Label ;
FallthroughStmt  = "fallthrough" ;
```

---

## 7. Source File & Package Grammar

```ebnf
SourceFile       = PackageClause ";" { ImportDecl ";" } { TopLevelDecl ";" } ;

PackageClause    = "package" PackageName ;
PackageName      = identifier ;

ImportDecl       = "import" ( ImportSpec | "(" { ImportSpec ";" } ")" ) ;
ImportSpec       = [ "." | PackageName ] ImportPath ;
ImportPath       = string_lit ;
```

---

## 8. Comparison: Full Go vs. Project Subset

| Language Feature | Full Go Language Specification | Robust Go Parser Project Subset |
|---|---|---|
| **Lexer & ASI** | Full UTF-8, Runes, Raw strings, Hex floats, ASI | Full UTF-8, Standard strings, Floats, Ints, exact ASI |
| **Expressions** | Full Pratt with 5 precedence levels + Bitwise/Shifts | Full Pratt with all 5 precedence levels + Bitwise/Shifts |
| **Control Flow** | `if`, `for`, `switch`, `type-switch`, `select` | `if`/`else`, `for` (3 forms: 3-clause, while, infinite) |
| **Functions** | Multi-return, Named returns, Generics, Methods | Single/multi parameter, Return types, Full body |
| **Types** | Structs, Interfaces, Channels, Maps, Generics | Primitives (`int`, `float`, `string`, `bool`), Identifiers |
| **Concurrency** | `go func()`, `chan<-`, `select`, `defer` | Sequential execution with AST Evaluator (`eval.go`) |
| **Error Handling**| Compiler abort or multi-pass errors | **Panic-Mode Recovery** (`synchronize()`) with Caret Previews |

---

## 9. References

- [The Go Programming Language Specification](https://go.dev/ref/spec)
- [Go 1.18 Generics Type Parameters Specification](https://go.dev/ref/spec#Type_parameter_declarations)
- [ISO/IEC 14977 EBNF Standard](https://www.iso.org/standard/27238.html)
