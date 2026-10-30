package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Value represents a runtime value during dynamic execution.
type Value struct {
	Kind       ValKind
	IntVal     int64
	FltVal     float64
	StrVal     string
	BoolVal    bool
	FnVal      *FuncDecl
	SliceVal   []Value
	StructVal  map[string]Value
	StructType string
	MapVal     map[string]Value
	IsReturn   bool
	IsBreak    bool
	IsContinue bool
}

type ValKind int

const (
	ValNil ValKind = iota
	ValInt
	ValFloat
	ValString
	ValBool
	ValFunc
	ValSlice
	ValStruct
	ValMap
)

func (v Value) String() string {
	switch v.Kind {
	case ValInt:
		return strconv.FormatInt(v.IntVal, 10)
	case ValFloat:
		return strconv.FormatFloat(v.FltVal, 'g', -1, 64)
	case ValString:
		return v.StrVal
	case ValBool:
		return strconv.FormatBool(v.BoolVal)
	case ValFunc:
		if v.FnVal != nil && v.FnVal.Name != nil {
			return fmt.Sprintf("<func %s>", v.FnVal.Name.Name)
		}
		return "<func>"
	case ValSlice:
		var items []string
		for _, el := range v.SliceVal {
			items = append(items, el.String())
		}
		return "[" + strings.Join(items, " ") + "]"
	case ValStruct:
		var items []string
		for k, val := range v.StructVal {
			items = append(items, fmt.Sprintf("%s:%s", k, val.String()))
		}
		return "{" + strings.Join(items, " ") + "}"
	case ValMap:
		var items []string
		var keys []string
		for k := range v.MapVal {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			items = append(items, fmt.Sprintf("%s:%s", k, v.MapVal[k].String()))
		}
		return "map[" + strings.Join(items, " ") + "]"
	default:
		return "<nil>"
	}
}

func (v Value) IsTruthy() bool {
	switch v.Kind {
	case ValBool:
		return v.BoolVal
	case ValInt:
		return v.IntVal != 0
	case ValFloat:
		return v.FltVal != 0
	case ValString:
		return len(v.StrVal) > 0
	default:
		return false
	}
}

// Env represents a lexical scope environment for variable storage.
type Env struct {
	vars         map[string]Value
	parent       *Env
	stdout       *strings.Builder
	stdin        io.Reader
	directStdout io.Writer
	defers       []*CallExpr
}

func NewEnv(parent *Env) *Env {
	var sb *strings.Builder
	var stdin io.Reader
	var directStdout io.Writer
	if parent != nil {
		sb = parent.stdout
		stdin = parent.stdin
		directStdout = parent.directStdout
	} else {
		sb = &strings.Builder{}
		stdin = os.Stdin
	}
	env := &Env{
		vars:         make(map[string]Value),
		parent:       parent,
		stdout:       sb,
		stdin:        stdin,
		directStdout: directStdout,
	}
	if parent == nil {
		env.Define("true", Value{Kind: ValBool, BoolVal: true})
		env.Define("false", Value{Kind: ValBool, BoolVal: false})
		env.Define("nil", Value{Kind: ValNil})
	}
	return env
}

func (e *Env) WriteOut(s string) {
	if e.stdout != nil {
		e.stdout.WriteString(s)
	}
	if e.directStdout != nil {
		fmt.Fprint(e.directStdout, s)
	}
}

func (e *Env) Get(name string) (Value, bool) {
	if val, ok := e.vars[name]; ok {
		return val, true
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return Value{}, false
}

func (e *Env) Set(name string, val Value) {
	// If exists in current or parent scope, update it
	curr := e
	for curr != nil {
		if _, ok := curr.vars[name]; ok {
			curr.vars[name] = val
			return
		}
		curr = curr.parent
	}
	e.vars[name] = val
}

func (e *Env) Define(name string, val Value) {
	e.vars[name] = val
}

// RuntimeError represents an execution failure with an exact source position.
type RuntimeError struct {
	Pos     Position
	Message string
}

func (e *RuntimeError) Error() string {
	if e.Pos.Line > 0 {
		return fmt.Sprintf("runtime error at line %d, col %d: %s", e.Pos.Line, e.Pos.Col, e.Message)
	}
	return fmt.Sprintf("runtime error: %s", e.Message)
}

func runtimeErr(pos Position, format string, args ...any) *RuntimeError {
	return &RuntimeError{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
	}
}

func wrapRuntimeErr(err error, fallbackPos Position) error {
	if err == nil {
		return nil
	}
	if rErr, ok := err.(*RuntimeError); ok {
		if rErr.Pos.Line == 0 && fallbackPos.Line > 0 {
			rErr.Pos = fallbackPos
		}
		return rErr
	}
	return &RuntimeError{
		Pos:     fallbackPos,
		Message: err.Error(),
	}
}

// Interpreter dynamically evaluates Go AST nodes.
type Interpreter struct {
	GlobalEnv *Env
	methods   map[string]map[string]*FuncDecl // typeName -> methodName -> FuncDecl
}

func NewInterpreter() *Interpreter {
	return NewInterpreterWithIO(os.Stdin, nil)
}

func NewInterpreterWithIO(stdin io.Reader, directStdout io.Writer) *Interpreter {
	env := NewEnv(nil)
	if stdin != nil {
		env.stdin = stdin
	}
	env.directStdout = directStdout
	return &Interpreter{
		GlobalEnv: env,
		methods:   make(map[string]map[string]*FuncDecl),
	}
}

// Eval dynamically evaluates an AST Expression.
func (in *Interpreter) Eval(expr Expr, env *Env) (val Value, err error) {
	if expr == nil {
		return Value{}, nil
	}
	defer func() {
		if err != nil {
			err = wrapRuntimeErr(err, expr.Pos())
		}
	}()

	switch node := expr.(type) {
	case *BasicLit:
		switch node.Kind {
		case INT:
			if strings.HasPrefix(node.Value, "'") && strings.HasSuffix(node.Value, "'") && len(node.Value) >= 3 {
				content := node.Value[1 : len(node.Value)-1]
				if strings.HasPrefix(content, "\\") && len(content) > 1 {
					switch content[1] {
					case 'n':
						return Value{Kind: ValInt, IntVal: int64('\n')}, nil
					case 't':
						return Value{Kind: ValInt, IntVal: int64('\t')}, nil
					case 'r':
						return Value{Kind: ValInt, IntVal: int64('\r')}, nil
					case '\\':
						return Value{Kind: ValInt, IntVal: int64('\\')}, nil
					case '\'':
						return Value{Kind: ValInt, IntVal: int64('\'')}, nil
					}
				}
				runes := []rune(content)
				if len(runes) > 0 {
					return Value{Kind: ValInt, IntVal: int64(runes[0])}, nil
				}
			}
			cleanInt := strings.ReplaceAll(node.Value, "_", "")
			n, err := strconv.ParseInt(cleanInt, 0, 64)
			if err != nil {
				return Value{}, fmt.Errorf("invalid int literal: %s", node.Value)
			}
			return Value{Kind: ValInt, IntVal: n}, nil
		case FLOAT:
			cleanFlt := strings.ReplaceAll(node.Value, "_", "")
			f, err := strconv.ParseFloat(cleanFlt, 64)
			if err != nil {
				return Value{}, fmt.Errorf("invalid float literal: %s", node.Value)
			}
			return Value{Kind: ValFloat, FltVal: f}, nil
		case STRING:
			unquoted := node.Value
			if strings.HasPrefix(unquoted, "\"") && strings.HasSuffix(unquoted, "\"") {
				if s, err := strconv.Unquote(unquoted); err == nil {
					return Value{Kind: ValString, StrVal: s}, nil
				}
				unquoted = unquoted[1 : len(unquoted)-1]
			} else if strings.HasPrefix(unquoted, "`") && strings.HasSuffix(unquoted, "`") {
				unquoted = unquoted[1 : len(unquoted)-1]
			}
			return Value{Kind: ValString, StrVal: unquoted}, nil
		}

	case *ArrayType, *MapType:
		return Value{Kind: ValNil}, nil

	case *Ident:
		if node.Name == "_" {
			return Value{}, runtimeErr(node.Pos(), "cannot use _ as value")
		}
		switch node.Name {
		case "true":
			return Value{Kind: ValBool, BoolVal: true}, nil
		case "false":
			return Value{Kind: ValBool, BoolVal: false}, nil
		case "nil":
			return Value{Kind: ValNil}, nil
		}
		val, ok := env.Get(node.Name)
		if !ok {
			return Value{}, runtimeErr(node.Pos(), "undefined identifier: %s", node.Name)
		}
		return val, nil

	case *ParenExpr:
		return in.Eval(node.X, env)

	case *UnaryExpr:
		xVal, err := in.Eval(node.X, env)
		if err != nil {
			return Value{}, err
		}
		switch node.Op {
		case SUB:
			if xVal.Kind == ValInt {
				return Value{Kind: ValInt, IntVal: -xVal.IntVal}, nil
			}
			if xVal.Kind == ValFloat {
				return Value{Kind: ValFloat, FltVal: -xVal.FltVal}, nil
			}
			return Value{}, runtimeErr(node.Pos(), "invalid operand for unary -: %s", xVal)
		case ADD:
			return xVal, nil
		case AND:
			if ident, ok := node.X.(*Ident); ok {
				return Value{Kind: ValString, StrVal: ident.Name}, nil
			}
			return xVal, nil
		case NOT:
			return Value{Kind: ValBool, BoolVal: !xVal.IsTruthy()}, nil
		case XOR:
			if xVal.Kind == ValInt {
				return Value{Kind: ValInt, IntVal: ^xVal.IntVal}, nil
			}
			return Value{}, runtimeErr(node.Pos(), "invalid operand for unary ^: %s", xVal)
		}

	case *BinaryExpr:
		xVal, err := in.Eval(node.X, env)
		if err != nil {
			return Value{}, err
		}
		// Short-circuit logical operators
		if node.Op == LOR {
			if xVal.IsTruthy() {
				return Value{Kind: ValBool, BoolVal: true}, nil
			}
			yVal, err := in.Eval(node.Y, env)
			if err != nil {
				return Value{}, err
			}
			return Value{Kind: ValBool, BoolVal: yVal.IsTruthy()}, nil
		}
		if node.Op == LAND {
			if !xVal.IsTruthy() {
				return Value{Kind: ValBool, BoolVal: false}, nil
			}
			yVal, err := in.Eval(node.Y, env)
			if err != nil {
				return Value{}, err
			}
			return Value{Kind: ValBool, BoolVal: yVal.IsTruthy()}, nil
		}

		yVal, err := in.Eval(node.Y, env)
		if err != nil {
			return Value{}, err
		}

		return in.evalBinary(node.Op, xVal, yVal)

	case *CallExpr:
		// Check for method calls on structs: obj.Method(...)
		if sel, ok := node.Fun.(*SelectorExpr); ok {
			receiverVal, err := in.Eval(sel.X, env)
			if err == nil && receiverVal.Kind == ValStruct {
				if typeMethods, ok := in.methods[receiverVal.StructType]; ok {
					if fn, ok := typeMethods[sel.Sel.Name]; ok {
						methodEnv := NewEnv(in.GlobalEnv)
						if fn.Recv != nil && len(fn.Recv.Names) > 0 {
							methodEnv.Define(fn.Recv.Names[0].Name, receiverVal)
						}
						var paramNames []string
						for _, param := range fn.Params {
							for _, name := range param.Names {
								paramNames = append(paramNames, name.Name)
							}
						}
						for i, name := range paramNames {
							if i < len(node.Args) {
								argVal, err := in.Eval(node.Args[i], env)
								if err != nil {
									return Value{}, err
								}
								methodEnv.Define(name, argVal)
							} else {
								methodEnv.Define(name, Value{})
							}
						}
						retVal, err := in.ExecBlock(fn.Body, methodEnv)
						for i := len(methodEnv.defers) - 1; i >= 0; i-- {
							_, _ = in.Eval(methodEnv.defers[i], methodEnv)
						}
						if err != nil {
							return Value{}, err
						}
						retVal.IsReturn = false
						return retVal, nil
					}
				}
			}

			// Standard library packages
			if pkg, ok := sel.X.(*Ident); ok {
				switch pkg.Name {
				case "fmt":
					switch sel.Sel.Name {
					case "Print", "Println":
						var parts []string
						for _, arg := range node.Args {
							v, err := in.Eval(arg, env)
							if err != nil {
								return Value{}, err
							}
							parts = append(parts, v.String())
						}
						sep := ""
						if sel.Sel.Name == "Println" {
							sep = " "
						}
						out := strings.Join(parts, sep)
						if sel.Sel.Name == "Println" {
							out += "\n"
						}
						env.WriteOut(out)
						return Value{}, nil
					case "Printf":
						if len(node.Args) > 0 {
							fmtVal, err := in.Eval(node.Args[0], env)
							if err != nil {
								return Value{}, err
							}
							var anyArgs []any
							for _, arg := range node.Args[1:] {
								v, err := in.Eval(arg, env)
								if err != nil {
									return Value{}, err
								}
								switch v.Kind {
								case ValInt:
									anyArgs = append(anyArgs, v.IntVal)
								case ValFloat:
									anyArgs = append(anyArgs, v.FltVal)
								case ValString:
									anyArgs = append(anyArgs, v.StrVal)
								case ValBool:
									anyArgs = append(anyArgs, v.BoolVal)
								default:
									anyArgs = append(anyArgs, v.String())
								}
							}
							out := fmt.Sprintf(fmtVal.StrVal, anyArgs...)
							env.WriteOut(out)
							return Value{}, nil
						}
					case "Sprint", "Sprintln", "Sprintf":
						if sel.Sel.Name == "Sprintf" && len(node.Args) > 0 {
							fmtVal, err := in.Eval(node.Args[0], env)
							if err != nil {
								return Value{}, err
							}
							var anyArgs []any
							for _, arg := range node.Args[1:] {
								v, err := in.Eval(arg, env)
								if err != nil {
									return Value{}, err
								}
								switch v.Kind {
								case ValInt:
									anyArgs = append(anyArgs, v.IntVal)
								case ValFloat:
									anyArgs = append(anyArgs, v.FltVal)
								case ValString:
									anyArgs = append(anyArgs, v.StrVal)
								case ValBool:
									anyArgs = append(anyArgs, v.BoolVal)
								default:
									anyArgs = append(anyArgs, v.String())
								}
							}
							out := fmt.Sprintf(fmtVal.StrVal, anyArgs...)
							return Value{Kind: ValString, StrVal: out}, nil
						}
						var parts []string
						for _, arg := range node.Args {
							v, err := in.Eval(arg, env)
							if err != nil {
								return Value{}, err
							}
							parts = append(parts, v.String())
						}
						sep := ""
						if sel.Sel.Name == "Sprintln" {
							sep = " "
						}
						out := strings.Join(parts, sep)
						if sel.Sel.Name == "Sprintln" {
							out += "\n"
						}
						return Value{Kind: ValString, StrVal: out}, nil
					case "Scan", "Scanln", "Scanf":
						argsToScan := node.Args
						if sel.Sel.Name == "Scanf" && len(argsToScan) > 1 {
							argsToScan = argsToScan[1:]
						}
						for _, arg := range argsToScan {
							var targetName string
							if u, ok := arg.(*UnaryExpr); ok && u.Op == AND {
								if target, ok := u.X.(*Ident); ok {
									targetName = target.Name
								}
							} else if ident, ok := arg.(*Ident); ok {
								targetName = ident.Name
							}

							if targetName == "" {
								continue
							}

							if env.stdin == nil {
								return Value{}, fmt.Errorf("fmt.%s: stdin is not configured", sel.Sel.Name)
							}

							var token string
							_, err := fmt.Fscan(env.stdin, &token)
							if err != nil {
								return Value{}, fmt.Errorf("fmt.%s error: %w", sel.Sel.Name, err)
							}

							if env.directStdout == nil {
								env.WriteOut(token + "\n")
							} else {
								env.stdout.WriteString(token + "\n")
								if f, ok := env.stdin.(*os.File); ok {
									stat, err := f.Stat()
									if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
										fmt.Fprintln(env.directStdout, token)
									}
								}
							}

							// Type conversion and assignment
							curr, exists := env.Get(targetName)
							if exists && curr.Kind == ValFloat {
								if f, err := strconv.ParseFloat(token, 64); err == nil {
									env.Set(targetName, Value{Kind: ValFloat, FltVal: f})
									continue
								}
							} else if exists && curr.Kind == ValBool {
								if b, err := strconv.ParseBool(token); err == nil {
									env.Set(targetName, Value{Kind: ValBool, BoolVal: b})
									continue
								}
							} else if exists && curr.Kind == ValString {
								env.Set(targetName, Value{Kind: ValString, StrVal: token})
								continue
							}

							// Default / integer detection
							if n, err := strconv.ParseInt(token, 0, 64); err == nil {
								env.Set(targetName, Value{Kind: ValInt, IntVal: n})
							} else if f, err := strconv.ParseFloat(token, 64); err == nil {
								env.Set(targetName, Value{Kind: ValFloat, FltVal: f})
							} else if b, err := strconv.ParseBool(token); err == nil {
								env.Set(targetName, Value{Kind: ValBool, BoolVal: b})
							} else {
								env.Set(targetName, Value{Kind: ValString, StrVal: token})
							}
						}
						return Value{}, nil
					}
				case "math":
					var args []float64
					for _, arg := range node.Args {
						v, err := in.Eval(arg, env)
						if err != nil {
							return Value{}, err
						}
						if v.Kind == ValFloat {
							args = append(args, v.FltVal)
						} else {
							args = append(args, float64(v.IntVal))
						}
					}
					switch sel.Sel.Name {
					case "Sqrt":
						if len(args) > 0 {
							return Value{Kind: ValFloat, FltVal: math.Sqrt(args[0])}, nil
						}
					case "Abs":
						if len(args) > 0 {
							return Value{Kind: ValFloat, FltVal: math.Abs(args[0])}, nil
						}
					case "Pow":
						if len(args) >= 2 {
							return Value{Kind: ValFloat, FltVal: math.Pow(args[0], args[1])}, nil
						}
					case "Min":
						if len(args) >= 2 {
							return Value{Kind: ValFloat, FltVal: math.Min(args[0], args[1])}, nil
						}
					case "Max":
						if len(args) >= 2 {
							return Value{Kind: ValFloat, FltVal: math.Max(args[0], args[1])}, nil
						}
					case "Floor":
						if len(args) > 0 {
							return Value{Kind: ValFloat, FltVal: math.Floor(args[0])}, nil
						}
					case "Ceil":
						if len(args) > 0 {
							return Value{Kind: ValFloat, FltVal: math.Ceil(args[0])}, nil
						}
					case "Round":
						if len(args) > 0 {
							return Value{Kind: ValFloat, FltVal: math.Round(args[0])}, nil
						}
					}
				case "strings":
					var sArgs []string
					for _, arg := range node.Args {
						v, err := in.Eval(arg, env)
						if err != nil {
							return Value{}, err
						}
						sArgs = append(sArgs, v.String())
					}
					switch sel.Sel.Name {
					case "Contains":
						if len(sArgs) >= 2 {
							return Value{Kind: ValBool, BoolVal: strings.Contains(sArgs[0], sArgs[1])}, nil
						}
					case "HasPrefix":
						if len(sArgs) >= 2 {
							return Value{Kind: ValBool, BoolVal: strings.HasPrefix(sArgs[0], sArgs[1])}, nil
						}
					case "HasSuffix":
						if len(sArgs) >= 2 {
							return Value{Kind: ValBool, BoolVal: strings.HasSuffix(sArgs[0], sArgs[1])}, nil
						}
					case "ToUpper":
						if len(sArgs) > 0 {
							return Value{Kind: ValString, StrVal: strings.ToUpper(sArgs[0])}, nil
						}
					case "ToLower":
						if len(sArgs) > 0 {
							return Value{Kind: ValString, StrVal: strings.ToLower(sArgs[0])}, nil
						}
					case "TrimSpace":
						if len(sArgs) > 0 {
							return Value{Kind: ValString, StrVal: strings.TrimSpace(sArgs[0])}, nil
						}
					case "Split":
						if len(sArgs) >= 2 {
							parts := strings.Split(sArgs[0], sArgs[1])
							var vals []Value
							for _, p := range parts {
								vals = append(vals, Value{Kind: ValString, StrVal: p})
							}
							return Value{Kind: ValSlice, SliceVal: vals}, nil
						}
					}
				case "strconv":
					if sel.Sel.Name == "Itoa" && len(node.Args) > 0 {
						v, _ := in.Eval(node.Args[0], env)
						return Value{Kind: ValString, StrVal: strconv.FormatInt(v.IntVal, 10)}, nil
					}
					if sel.Sel.Name == "Atoi" && len(node.Args) > 0 {
						v, _ := in.Eval(node.Args[0], env)
						n, _ := strconv.ParseInt(v.String(), 0, 64)
						return Value{Kind: ValInt, IntVal: n}, nil
					}
				}
			}
		}

		if ident, ok := node.Fun.(*Ident); ok {
			switch ident.Name {
			case "print", "println":
				var parts []string
				for _, arg := range node.Args {
					v, err := in.Eval(arg, env)
					if err != nil {
						return Value{}, err
					}
					parts = append(parts, v.String())
				}
				sep := " "
				out := strings.Join(parts, sep)
				if ident.Name == "println" {
					out += "\n"
				}
				env.WriteOut(out)
				return Value{}, nil
			case "len":
				if len(node.Args) > 0 {
					v, err := in.Eval(node.Args[0], env)
					if err != nil {
						return Value{}, err
					}
					if v.Kind == ValString {
						return Value{Kind: ValInt, IntVal: int64(len(v.StrVal))}, nil
					}
					if v.Kind == ValSlice {
						return Value{Kind: ValInt, IntVal: int64(len(v.SliceVal))}, nil
					}
					if v.Kind == ValMap {
						return Value{Kind: ValInt, IntVal: int64(len(v.MapVal))}, nil
					}
				}
				return Value{Kind: ValInt, IntVal: 0}, nil
			case "cap":
				if len(node.Args) > 0 {
					v, err := in.Eval(node.Args[0], env)
					if err != nil {
						return Value{}, err
					}
					if v.Kind == ValSlice {
						return Value{Kind: ValInt, IntVal: int64(len(v.SliceVal))}, nil
					}
				}
				return Value{Kind: ValInt, IntVal: 0}, nil
			case "make":
				if len(node.Args) > 0 {
					if _, ok := node.Args[0].(*MapType); ok {
						return Value{Kind: ValMap, MapVal: make(map[string]Value)}, nil
					}
					if id, ok := node.Args[0].(*Ident); ok && id.Name == "map" {
						return Value{Kind: ValMap, MapVal: make(map[string]Value)}, nil
					}
					size := int64(0)
					if len(node.Args) > 1 {
						sVal, err := in.Eval(node.Args[1], env)
						if err == nil && sVal.Kind == ValInt {
							size = sVal.IntVal
						}
					}
					items := make([]Value, size)
					for i := range items {
						items[i] = Value{Kind: ValInt, IntVal: 0}
					}
					return Value{Kind: ValSlice, SliceVal: items}, nil
				}
				return Value{Kind: ValSlice}, nil
			case "delete":
				if len(node.Args) >= 2 {
					mVal, err := in.Eval(node.Args[0], env)
					if err != nil {
						return Value{}, err
					}
					kVal, err := in.Eval(node.Args[1], env)
					if err != nil {
						return Value{}, err
					}
					if mVal.Kind == ValMap && mVal.MapVal != nil {
						delete(mVal.MapVal, kVal.String())
					}
				}
				return Value{}, nil
			case "copy":
				if len(node.Args) >= 2 {
					dstVal, err := in.Eval(node.Args[0], env)
					if err != nil {
						return Value{}, err
					}
					srcVal, err := in.Eval(node.Args[1], env)
					if err != nil {
						return Value{}, err
					}
					if dstVal.Kind == ValSlice && srcVal.Kind == ValSlice {
						n := copy(dstVal.SliceVal, srcVal.SliceVal)
						return Value{Kind: ValInt, IntVal: int64(n)}, nil
					}
				}
				return Value{Kind: ValInt, IntVal: 0}, nil
			case "panic":
				var msg string
				if len(node.Args) > 0 {
					v, _ := in.Eval(node.Args[0], env)
					msg = v.String()
				}
				return Value{}, runtimeErr(node.Pos(), "panic: %s", msg)
			case "recover":
				return Value{Kind: ValNil}, nil
			case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "byte", "rune":
				if len(node.Args) > 0 {
					v, err := in.Eval(node.Args[0], env)
					if err != nil {
						return Value{}, err
					}
					if v.Kind == ValInt {
						return v, nil
					}
					if v.Kind == ValFloat {
						return Value{Kind: ValInt, IntVal: int64(v.FltVal)}, nil
					}
					if v.Kind == ValString {
						if n, err := strconv.ParseInt(v.StrVal, 0, 64); err == nil {
							return Value{Kind: ValInt, IntVal: n}, nil
						}
					}
				}
				return Value{Kind: ValInt, IntVal: 0}, nil
			case "float32", "float64":
				if len(node.Args) > 0 {
					v, err := in.Eval(node.Args[0], env)
					if err != nil {
						return Value{}, err
					}
					if v.Kind == ValFloat {
						return v, nil
					}
					if v.Kind == ValInt {
						return Value{Kind: ValFloat, FltVal: float64(v.IntVal)}, nil
					}
					if v.Kind == ValString {
						if f, err := strconv.ParseFloat(v.StrVal, 64); err == nil {
							return Value{Kind: ValFloat, FltVal: f}, nil
						}
					}
				}
				return Value{Kind: ValFloat, FltVal: 0.0}, nil
			case "string":
				if len(node.Args) > 0 {
					v, err := in.Eval(node.Args[0], env)
					if err != nil {
						return Value{}, err
					}
					if v.Kind == ValInt {
						return Value{Kind: ValString, StrVal: string(rune(v.IntVal))}, nil
					}
					return Value{Kind: ValString, StrVal: v.String()}, nil
				}
				return Value{Kind: ValString, StrVal: ""}, nil
			case "bool":
				if len(node.Args) > 0 {
					v, err := in.Eval(node.Args[0], env)
					if err != nil {
						return Value{}, err
					}
					return Value{Kind: ValBool, BoolVal: v.IsTruthy()}, nil
				}
				return Value{Kind: ValBool, BoolVal: false}, nil
			case "append":
				if len(node.Args) >= 2 {
					sliceVal, err := in.Eval(node.Args[0], env)
					if err != nil {
						return Value{}, err
					}
					itemVal, err := in.Eval(node.Args[1], env)
					if err != nil {
						return Value{}, err
					}
					var newSlice []Value
					if sliceVal.Kind == ValSlice {
						newSlice = append(newSlice, sliceVal.SliceVal...)
					}
					newSlice = append(newSlice, itemVal)
					return Value{Kind: ValSlice, SliceVal: newSlice}, nil
				}
				return Value{Kind: ValSlice}, nil
			}
		}

		fnVal, err := in.Eval(node.Fun, env)
		if err != nil {
			return Value{}, err
		}
		if fnVal.Kind != ValFunc || fnVal.FnVal == nil {
			return Value{}, fmt.Errorf("cannot call non-function %s", fnVal)
		}

		fn := fnVal.FnVal
		callEnv := NewEnv(in.GlobalEnv)

		// Bind parameters
		var paramNames []string
		for _, param := range fn.Params {
			for _, name := range param.Names {
				paramNames = append(paramNames, name.Name)
			}
		}

		for i, name := range paramNames {
			if i < len(node.Args) {
				argVal, err := in.Eval(node.Args[i], env)
				if err != nil {
					return Value{}, err
				}
				callEnv.Define(name, argVal)
			} else {
				callEnv.Define(name, Value{})
			}
		}

		retVal, err := in.ExecBlock(fn.Body, callEnv)
		// Run defers in LIFO order
		for i := len(callEnv.defers) - 1; i >= 0; i-- {
			_, _ = in.Eval(callEnv.defers[i], callEnv)
		}
		if err != nil {
			return Value{}, err
		}
		retVal.IsReturn = false
		return retVal, nil

	case *CompositeLit:
		if _, isArray := node.Type.(*ArrayType); isArray {
			var sliceItems []Value
			for _, elt := range node.Elts {
				item, err := in.Eval(elt, env)
				if err != nil {
					return Value{}, err
				}
				sliceItems = append(sliceItems, item)
			}
			return Value{Kind: ValSlice, SliceVal: sliceItems}, nil
		}
		if _, isMap := node.Type.(*MapType); isMap {
			mapItems := make(map[string]Value)
			for _, elt := range node.Elts {
				if kv, ok := elt.(*KeyValueExpr); ok {
					kVal, err := in.Eval(kv.Key, env)
					if err != nil {
						return Value{}, err
					}
					vVal, err := in.Eval(kv.Value, env)
					if err != nil {
						return Value{}, err
					}
					mapItems[kVal.String()] = vVal
				}
			}
			return Value{Kind: ValMap, MapVal: mapItems}, nil
		}
		// Struct literal: e.g. Point{X: 10, Y: 20}
		structFields := make(map[string]Value)
		var typeName string
		if id, ok := node.Type.(*Ident); ok {
			typeName = id.Name
		}
		for _, elt := range node.Elts {
			if kv, ok := elt.(*KeyValueExpr); ok {
				if keyIdent, ok := kv.Key.(*Ident); ok {
					val, err := in.Eval(kv.Value, env)
					if err != nil {
						return Value{}, err
					}
					structFields[keyIdent.Name] = val
				}
			} else {
				val, err := in.Eval(elt, env)
				if err != nil {
					return Value{}, err
				}
				structFields[fmt.Sprintf("%d", len(structFields))] = val
			}
		}
		return Value{Kind: ValStruct, StructVal: structFields, StructType: typeName}, nil

	case *KeyValueExpr:
		return in.Eval(node.Value, env)

	case *SliceExpr:
		xVal, err := in.Eval(node.X, env)
		if err != nil {
			return Value{}, err
		}
		if xVal.Kind == ValSlice {
			low := int64(0)
			high := int64(len(xVal.SliceVal))
			if node.Low != nil {
				lVal, err := in.Eval(node.Low, env)
				if err != nil {
					return Value{}, err
				}
				low = lVal.IntVal
			}
			if node.High != nil {
				hVal, err := in.Eval(node.High, env)
				if err != nil {
					return Value{}, err
				}
				high = hVal.IntVal
			}
			if low < 0 {
				low = 0
			}
			if high > int64(len(xVal.SliceVal)) {
				high = int64(len(xVal.SliceVal))
			}
			if low > high {
				low = high
			}
			return Value{Kind: ValSlice, SliceVal: xVal.SliceVal[low:high]}, nil
		}
		if xVal.Kind == ValString {
			low := int64(0)
			high := int64(len(xVal.StrVal))
			if node.Low != nil {
				lVal, _ := in.Eval(node.Low, env)
				low = lVal.IntVal
			}
			if node.High != nil {
				hVal, _ := in.Eval(node.High, env)
				high = hVal.IntVal
			}
			if low < 0 {
				low = 0
			}
			if high > int64(len(xVal.StrVal)) {
				high = int64(len(xVal.StrVal))
			}
			if low > high {
				low = high
			}
			return Value{Kind: ValString, StrVal: xVal.StrVal[low:high]}, nil
		}
		return Value{}, fmt.Errorf("cannot slice non-slice/string %s", xVal)

	case *IndexExpr:
		xVal, err := in.Eval(node.X, env)
		if err != nil {
			return Value{}, err
		}
		idxVal, err := in.Eval(node.Index, env)
		if err != nil {
			return Value{}, err
		}
		if xVal.Kind == ValSlice {
			idx := idxVal.IntVal
			if idx < 0 || idx >= int64(len(xVal.SliceVal)) {
				return Value{}, fmt.Errorf("index out of range [%d] with length %d", idx, len(xVal.SliceVal))
			}
			return xVal.SliceVal[idx], nil
		}
		if xVal.Kind == ValString {
			idx := idxVal.IntVal
			if idx < 0 || idx >= int64(len(xVal.StrVal)) {
				return Value{}, fmt.Errorf("index out of range [%d] with length %d", idx, len(xVal.StrVal))
			}
			return Value{Kind: ValInt, IntVal: int64(xVal.StrVal[idx])}, nil
		}
		if xVal.Kind == ValMap && xVal.MapVal != nil {
			k := idxVal.String()
			if v, ok := xVal.MapVal[k]; ok {
				return v, nil
			}
			return Value{Kind: ValInt, IntVal: 0}, nil
		}
		return Value{}, fmt.Errorf("cannot index non-indexable %s", xVal)

	case *SelectorExpr:
		if id, ok := node.X.(*Ident); ok && id.Name == "math" {
			switch node.Sel.Name {
			case "Pi":
				return Value{Kind: ValFloat, FltVal: math.Pi}, nil
			case "E":
				return Value{Kind: ValFloat, FltVal: math.E}, nil
			}
		}
		xVal, err := in.Eval(node.X, env)
		if err != nil {
			return Value{}, err
		}
		if xVal.Kind == ValStruct && xVal.StructVal != nil {
			if v, ok := xVal.StructVal[node.Sel.Name]; ok {
				return v, nil
			}
			if typeMethods, ok := in.methods[xVal.StructType]; ok {
				if fn, ok := typeMethods[node.Sel.Name]; ok {
					return Value{Kind: ValFunc, FnVal: fn}, nil
				}
			}
			return Value{}, nil
		}
		return Value{}, fmt.Errorf("field %s not found on %s", node.Sel.Name, xVal)
	}

	return Value{}, fmt.Errorf("unsupported expression: %T", expr)
}

func (in *Interpreter) evalBinary(op TokenType, x, y Value) (Value, error) {
	// String concatenation
	if op == ADD && (x.Kind == ValString || y.Kind == ValString) {
		return Value{Kind: ValString, StrVal: x.String() + y.String()}, nil
	}

	// Floats
	if x.Kind == ValFloat || y.Kind == ValFloat {
		xf := x.FltVal
		if x.Kind == ValInt {
			xf = float64(x.IntVal)
		}
		yf := y.FltVal
		if y.Kind == ValInt {
			yf = float64(y.IntVal)
		}
		switch op {
		case ADD:
			return Value{Kind: ValFloat, FltVal: xf + yf}, nil
		case SUB:
			return Value{Kind: ValFloat, FltVal: xf - yf}, nil
		case MUL:
			return Value{Kind: ValFloat, FltVal: xf * yf}, nil
		case QUO:
			if yf == 0 {
				return Value{}, fmt.Errorf("division by zero")
			}
			return Value{Kind: ValFloat, FltVal: xf / yf}, nil
		case EQL:
			return Value{Kind: ValBool, BoolVal: xf == yf}, nil
		case NEQ:
			return Value{Kind: ValBool, BoolVal: xf != yf}, nil
		case LSS:
			return Value{Kind: ValBool, BoolVal: xf < yf}, nil
		case LEQ:
			return Value{Kind: ValBool, BoolVal: xf <= yf}, nil
		case GTR:
			return Value{Kind: ValBool, BoolVal: xf > yf}, nil
		case GEQ:
			return Value{Kind: ValBool, BoolVal: xf >= yf}, nil
		}
	}

	// Integers
	if x.Kind == ValInt && y.Kind == ValInt {
		xi := x.IntVal
		yi := y.IntVal
		switch op {
		case ADD:
			return Value{Kind: ValInt, IntVal: xi + yi}, nil
		case SUB:
			return Value{Kind: ValInt, IntVal: xi - yi}, nil
		case MUL:
			return Value{Kind: ValInt, IntVal: xi * yi}, nil
		case QUO:
			if yi == 0 {
				return Value{}, fmt.Errorf("integer division by zero")
			}
			return Value{Kind: ValInt, IntVal: xi / yi}, nil
		case REM:
			if yi == 0 {
				return Value{}, fmt.Errorf("integer division by zero")
			}
			return Value{Kind: ValInt, IntVal: xi % yi}, nil
		case AND:
			return Value{Kind: ValInt, IntVal: xi & yi}, nil
		case OR:
			return Value{Kind: ValInt, IntVal: xi | yi}, nil
		case XOR:
			return Value{Kind: ValInt, IntVal: xi ^ yi}, nil
		case SHL:
			return Value{Kind: ValInt, IntVal: xi << uint64(yi)}, nil
		case SHR:
			return Value{Kind: ValInt, IntVal: xi >> uint64(yi)}, nil
		case AND_NOT:
			return Value{Kind: ValInt, IntVal: xi &^ yi}, nil
		case EQL:
			return Value{Kind: ValBool, BoolVal: xi == yi}, nil
		case NEQ:
			return Value{Kind: ValBool, BoolVal: xi != yi}, nil
		case LSS:
			return Value{Kind: ValBool, BoolVal: xi < yi}, nil
		case LEQ:
			return Value{Kind: ValBool, BoolVal: xi <= yi}, nil
		case GTR:
			return Value{Kind: ValBool, BoolVal: xi > yi}, nil
		case GEQ:
			return Value{Kind: ValBool, BoolVal: xi >= yi}, nil
		}
	}

	// Booleans
	if x.Kind == ValBool && y.Kind == ValBool {
		switch op {
		case EQL:
			return Value{Kind: ValBool, BoolVal: x.BoolVal == y.BoolVal}, nil
		case NEQ:
			return Value{Kind: ValBool, BoolVal: x.BoolVal != y.BoolVal}, nil
		}
	}

	return Value{}, fmt.Errorf("invalid binary operation %s on %s and %s", op, x, y)
}

// ExecStmt executes a statement dynamically.
func (in *Interpreter) ExecStmt(stmt Stmt, env *Env) (val Value, err error) {
	if stmt == nil {
		return Value{}, nil
	}
	defer func() {
		if err != nil {
			err = wrapRuntimeErr(err, stmt.Pos())
		}
	}()

	switch s := stmt.(type) {
	case *ImportDecl:
		return Value{}, nil

	case *IncDecStmt:
		if ident, ok := s.X.(*Ident); ok {
			curr, ok := env.Get(ident.Name)
			if ok && curr.Kind == ValInt {
				if s.Tok == INC {
					env.Set(ident.Name, Value{Kind: ValInt, IntVal: curr.IntVal + 1})
				} else {
					env.Set(ident.Name, Value{Kind: ValInt, IntVal: curr.IntVal - 1})
				}
			}
		}
		return Value{}, nil

	case *VarDecl:
		if len(s.Group) > 0 {
			for _, subDecl := range s.Group {
				if _, err := in.ExecStmt(subDecl, env); err != nil {
					return Value{}, err
				}
			}
			return Value{}, nil
		}
		val := Value{}
		if s.Value != nil {
			var err error
			val, err = in.Eval(s.Value, env)
			if err != nil {
				return Value{}, err
			}
		} else if s.Type != nil {
			if ident, ok := s.Type.(*Ident); ok {
				switch ident.Name {
				case "int", "int8", "int16", "int32", "int64":
					val = Value{Kind: ValInt, IntVal: 0}
				case "float32", "float64":
					val = Value{Kind: ValFloat, FltVal: 0.0}
				case "string":
					val = Value{Kind: ValString, StrVal: ""}
				case "bool":
					val = Value{Kind: ValBool, BoolVal: false}
				}
			} else if _, ok := s.Type.(*ArrayType); ok {
				val = Value{Kind: ValSlice, SliceVal: nil}
			} else if _, ok := s.Type.(*MapType); ok {
				val = Value{Kind: ValMap, MapVal: make(map[string]Value)}
			}
		}
		if len(s.Names) > 0 {
			for i, name := range s.Names {
				v := val
				if i < len(s.Values) {
					evalVal, err := in.Eval(s.Values[i], env)
					if err != nil {
						return Value{}, err
					}
					v = evalVal
				}
				env.Define(name.Name, v)
			}
		} else if s.Name != nil {
			env.Define(s.Name.Name, val)
		}
		return Value{}, nil

	case *AssignStmt:
		if len(s.LhsList) > 1 || len(s.RhsList) > 1 {
			var rhsVals []Value
			if len(s.RhsList) == 1 {
				singleVal, err := in.Eval(s.RhsList[0], env)
				if err != nil {
					return Value{}, err
				}
				if singleVal.Kind == ValSlice {
					rhsVals = singleVal.SliceVal
				} else {
					rhsVals = []Value{singleVal}
				}
			} else {
				for _, rhsExpr := range s.RhsList {
					rVal, err := in.Eval(rhsExpr, env)
					if err != nil {
						return Value{}, err
					}
					rhsVals = append(rhsVals, rVal)
				}
			}
			for i, lhs := range s.LhsList {
				var rVal Value
				if i < len(rhsVals) {
					rVal = rhsVals[i]
				}
				if ident, ok := lhs.(*Ident); ok {
					if ident.Name == "_" {
						continue
					}
					if s.Tok == DEFINE {
						env.Define(ident.Name, rVal)
					} else {
						env.Set(ident.Name, rVal)
					}
				} else if idxExpr, ok := lhs.(*IndexExpr); ok {
					if ident, ok := idxExpr.X.(*Ident); ok {
						curr, ok := env.Get(ident.Name)
						if ok && curr.Kind == ValSlice {
							idxVal, err := in.Eval(idxExpr.Index, env)
							if err == nil && idxVal.Kind == ValInt {
								idx := idxVal.IntVal
								if idx >= 0 && idx < int64(len(curr.SliceVal)) {
									curr.SliceVal[idx] = rVal
									env.Set(ident.Name, curr)
								}
							}
						} else if ok && curr.Kind == ValMap && curr.MapVal != nil {
							idxVal, err := in.Eval(idxExpr.Index, env)
							if err == nil {
								curr.MapVal[idxVal.String()] = rVal
								env.Set(ident.Name, curr)
							}
						}
					}
				} else if sel, ok := lhs.(*SelectorExpr); ok {
					if ident, ok := sel.X.(*Ident); ok {
						curr, ok := env.Get(ident.Name)
						if ok && curr.Kind == ValStruct && curr.StructVal != nil {
							curr.StructVal[sel.Sel.Name] = rVal
							env.Set(ident.Name, curr)
						}
					}
				}
			}
			return Value{}, nil
		}

		rhsVal, err := in.Eval(s.Rhs, env)
		if err != nil {
			return Value{}, err
		}
		if ident, ok := s.Lhs.(*Ident); ok {
			if ident.Name == "_" {
				return Value{}, nil
			}
			switch s.Tok {
			case DEFINE:
				env.Define(ident.Name, rhsVal)
			case ASSIGN:
				env.Set(ident.Name, rhsVal)
			case ADD_ASSIGN:
				curr, _ := env.Get(ident.Name)
				newVal, err := in.evalBinary(ADD, curr, rhsVal)
				if err != nil {
					return Value{}, err
				}
				env.Set(ident.Name, newVal)
			case SUB_ASSIGN:
				curr, _ := env.Get(ident.Name)
				newVal, err := in.evalBinary(SUB, curr, rhsVal)
				if err != nil {
					return Value{}, err
				}
				env.Set(ident.Name, newVal)
			case MUL_ASSIGN:
				curr, _ := env.Get(ident.Name)
				newVal, err := in.evalBinary(MUL, curr, rhsVal)
				if err != nil {
					return Value{}, err
				}
				env.Set(ident.Name, newVal)
			case QUO_ASSIGN:
				curr, _ := env.Get(ident.Name)
				newVal, err := in.evalBinary(QUO, curr, rhsVal)
				if err != nil {
					return Value{}, err
				}
				env.Set(ident.Name, newVal)
			case AND_ASSIGN, OR_ASSIGN, XOR_ASSIGN, SHL_ASSIGN, SHR_ASSIGN, AND_NOT_ASSIGN:
				var binOp TokenType
				switch s.Tok {
				case AND_ASSIGN:
					binOp = AND
				case OR_ASSIGN:
					binOp = OR
				case XOR_ASSIGN:
					binOp = XOR
				case SHL_ASSIGN:
					binOp = SHL
				case SHR_ASSIGN:
					binOp = SHR
				case AND_NOT_ASSIGN:
					binOp = AND_NOT
				}
				curr, _ := env.Get(ident.Name)
				newVal, err := in.evalBinary(binOp, curr, rhsVal)
				if err != nil {
					return Value{}, err
				}
				env.Set(ident.Name, newVal)
			}
			return Value{}, nil
		}
		if sel, ok := s.Lhs.(*SelectorExpr); ok {
			if ident, ok := sel.X.(*Ident); ok {
				curr, ok := env.Get(ident.Name)
				if ok && curr.Kind == ValStruct && curr.StructVal != nil {
					curr.StructVal[sel.Sel.Name] = rhsVal
					env.Set(ident.Name, curr)
					return Value{}, nil
				}
			}
		}
		if idxExpr, ok := s.Lhs.(*IndexExpr); ok {
			if ident, ok := idxExpr.X.(*Ident); ok {
				curr, ok := env.Get(ident.Name)
				if ok && curr.Kind == ValSlice {
					idxVal, err := in.Eval(idxExpr.Index, env)
					if err == nil && idxVal.Kind == ValInt {
						idx := idxVal.IntVal
						if idx >= 0 && idx < int64(len(curr.SliceVal)) {
							curr.SliceVal[idx] = rhsVal
							env.Set(ident.Name, curr)
							return Value{}, nil
						}
					}
				}
				if ok && curr.Kind == ValMap && curr.MapVal != nil {
					idxVal, err := in.Eval(idxExpr.Index, env)
					if err == nil {
						curr.MapVal[idxVal.String()] = rhsVal
						env.Set(ident.Name, curr)
						return Value{}, nil
					}
				}
			}
		}
		return Value{}, nil

	case *TypeDecl:
		return Value{}, nil

	case *ConstDecl:
		if len(s.Group) > 0 {
			for i, subDecl := range s.Group {
				env.Define("iota", Value{Kind: ValInt, IntVal: int64(i)})
				if _, err := in.ExecStmt(subDecl, env); err != nil {
					delete(env.vars, "iota")
					return Value{}, err
				}
			}
			delete(env.vars, "iota")
			return Value{}, nil
		}
		for i, name := range s.Names {
			val := Value{}
			if i < len(s.Values) {
				evalVal, err := in.Eval(s.Values[i], env)
				if err != nil {
					return Value{}, err
				}
				val = evalVal
			}
			env.Define(name.Name, val)
		}
		return Value{}, nil

	case *DeferStmt:
		env.defers = append(env.defers, s.Call)
		return Value{}, nil

	case *SwitchStmt:
		if s.Init != nil {
			if _, err := in.ExecStmt(s.Init, env); err != nil {
				return Value{}, err
			}
		}
		tagVal := Value{Kind: ValBool, BoolVal: true}
		if s.Tag != nil {
			v, err := in.Eval(s.Tag, env)
			if err != nil {
				return Value{}, err
			}
			tagVal = v
		}

		matched := false
		var defaultClause *CaseClause

		for _, clause := range s.Body {
			if len(clause.List) == 0 {
				defaultClause = clause
				continue
			}
			clauseMatched := false
			for _, cond := range clause.List {
				condVal, err := in.Eval(cond, env)
				if err != nil {
					return Value{}, err
				}
				eqVal, err := in.evalBinary(EQL, tagVal, condVal)
				if err == nil && eqVal.BoolVal {
					clauseMatched = true
					break
				}
			}
			if clauseMatched {
				matched = true
				caseEnv := NewEnv(env)
				for _, stmt := range clause.Body {
					res, err := in.ExecStmt(stmt, caseEnv)
					if err != nil {
						return Value{}, err
					}
					if res.IsReturn {
						return res, nil
					}
					if res.IsBreak {
						break
					}
				}
				break
			}
		}

		if !matched && defaultClause != nil {
			caseEnv := NewEnv(env)
			for _, stmt := range defaultClause.Body {
				res, err := in.ExecStmt(stmt, caseEnv)
				if err != nil {
					return Value{}, err
				}
				if res.IsReturn {
					return res, nil
				}
				if res.IsBreak {
					break
				}
			}
		}
		return Value{}, nil

	case *RangeStmt:
		xVal, err := in.Eval(s.X, env)
		if err != nil {
			return Value{}, err
		}

		bindVar := func(target Expr, val Value, loopEnv *Env) {
			if target == nil {
				return
			}
			if id, ok := target.(*Ident); ok {
				if id.Name == "_" {
					return
				}
				if s.Tok == DEFINE {
					loopEnv.Define(id.Name, val)
				} else {
					env.Set(id.Name, val)
				}
			}
		}

		if xVal.Kind == ValSlice {
			for i, item := range xVal.SliceVal {
				loopEnv := NewEnv(env)
				bindVar(s.Key, Value{Kind: ValInt, IntVal: int64(i)}, loopEnv)
				bindVar(s.Value, item, loopEnv)
				res, err := in.ExecBlock(s.Body, loopEnv)
				if err != nil {
					return Value{}, err
				}
				if res.IsReturn {
					return res, nil
				}
				if res.IsBreak {
					break
				}
				if res.IsContinue {
					continue
				}
			}
			return Value{}, nil
		}

		if xVal.Kind == ValMap {
			var keys []string
			for k := range xVal.MapVal {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				item := xVal.MapVal[k]
				loopEnv := NewEnv(env)
				bindVar(s.Key, Value{Kind: ValString, StrVal: k}, loopEnv)
				bindVar(s.Value, item, loopEnv)
				res, err := in.ExecBlock(s.Body, loopEnv)
				if err != nil {
					return Value{}, err
				}
				if res.IsReturn {
					return res, nil
				}
				if res.IsBreak {
					break
				}
				if res.IsContinue {
					continue
				}
			}
			return Value{}, nil
		}

		if xVal.Kind == ValString {
			for i, r := range xVal.StrVal {
				loopEnv := NewEnv(env)
				bindVar(s.Key, Value{Kind: ValInt, IntVal: int64(i)}, loopEnv)
				bindVar(s.Value, Value{Kind: ValInt, IntVal: int64(r)}, loopEnv)
				res, err := in.ExecBlock(s.Body, loopEnv)
				if err != nil {
					return Value{}, err
				}
				if res.IsReturn {
					return res, nil
				}
				if res.IsBreak {
					break
				}
				if res.IsContinue {
					continue
				}
			}
			return Value{}, nil
		}

		if xVal.Kind == ValInt {
			for i := int64(0); i < xVal.IntVal; i++ {
				loopEnv := NewEnv(env)
				bindVar(s.Key, Value{Kind: ValInt, IntVal: i}, loopEnv)
				res, err := in.ExecBlock(s.Body, loopEnv)
				if err != nil {
					return Value{}, err
				}
				if res.IsReturn {
					return res, nil
				}
				if res.IsBreak {
					break
				}
				if res.IsContinue {
					continue
				}
			}
			return Value{}, nil
		}
		return Value{}, nil

	case *ExprStmt:
		return in.Eval(s.X, env)

	case *BlockStmt:
		blockEnv := NewEnv(env)
		return in.ExecBlock(s, blockEnv)

	case *IfStmt:
		if s.Init != nil {
			_, err := in.ExecStmt(s.Init, env)
			if err != nil {
				return Value{}, err
			}
		}
		condVal, err := in.Eval(s.Cond, env)
		if err != nil {
			return Value{}, err
		}
		if condVal.IsTruthy() {
			return in.ExecStmt(s.Body, env)
		} else if s.Else != nil {
			return in.ExecStmt(s.Else, env)
		}
		return Value{}, nil

	case *ForStmt:
		forEnv := NewEnv(env)
		if s.Init != nil {
			_, err := in.ExecStmt(s.Init, forEnv)
			if err != nil {
				return Value{}, err
			}
		}
		iterCount := 0
		const maxIterations = 1000000
		for {
			iterCount++
			if iterCount > maxIterations {
				return Value{}, fmt.Errorf("loop iteration limit exceeded (%d iterations) - possible infinite loop", maxIterations)
			}
			if s.Cond != nil {
				condVal, err := in.Eval(s.Cond, forEnv)
				if err != nil {
					return Value{}, err
				}
				if !condVal.IsTruthy() {
					break
				}
			}
			retVal, err := in.ExecStmt(s.Body, forEnv)
			if err != nil {
				return Value{}, err
			}
			if retVal.IsReturn {
				return retVal, nil
			}
			if retVal.IsBreak {
				break
			}
			if s.Post != nil {
				_, err := in.ExecStmt(s.Post, forEnv)
				if err != nil {
					return Value{}, err
				}
			}
		}
		return Value{}, nil

	case *BranchStmt:
		if s.Tok == BREAK {
			return Value{IsBreak: true}, nil
		}
		if s.Tok == CONTINUE {
			return Value{IsContinue: true}, nil
		}
		return Value{}, nil

	case *ReturnStmt:
		if len(s.Results) > 1 {
			vals := make([]Value, len(s.Results))
			for i, r := range s.Results {
				v, err := in.Eval(r, env)
				if err != nil {
					return Value{}, err
				}
				vals[i] = v
			}
			return Value{Kind: ValSlice, SliceVal: vals, IsReturn: true}, nil
		} else if len(s.Results) == 1 {
			val, err := in.Eval(s.Results[0], env)
			if err != nil {
				return Value{}, err
			}
			val.IsReturn = true
			return val, nil
		}
		return Value{IsReturn: true}, nil
	}

	return Value{}, nil
}

func (in *Interpreter) ExecBlock(block *BlockStmt, env *Env) (Value, error) {
	if block == nil {
		return Value{}, nil
	}
	for _, s := range block.List {
		val, err := in.ExecStmt(s, env)
		if err != nil {
			return Value{}, err
		}
		if val.IsReturn || val.IsBreak || val.IsContinue {
			return val, nil
		}
	}
	return Value{}, nil
}

// ExecuteFile registers top-level declarations and executes main() or top-level statements.
func (in *Interpreter) ExecuteFile(file *File) (Value, string, error) {
	if file == nil {
		return Value{}, "", nil
	}

	// Register declarations
	for _, d := range file.Decls {
		switch decl := d.(type) {
		case *ConstDecl:
			_, err := in.ExecStmt(decl, in.GlobalEnv)
			if err != nil {
				return Value{}, in.GlobalEnv.stdout.String(), err
			}
		case *VarDecl:
			_, err := in.ExecStmt(decl, in.GlobalEnv)
			if err != nil {
				return Value{}, in.GlobalEnv.stdout.String(), err
			}
		case *FuncDecl:
			if decl.Recv != nil {
				var typeName string
				if id, ok := decl.Recv.Type.(*Ident); ok {
					typeName = id.Name
				} else if u, ok := decl.Recv.Type.(*UnaryExpr); ok && u.Op == MUL {
					if id, ok := u.X.(*Ident); ok {
						typeName = id.Name
					}
				}
				if typeName != "" && decl.Name != nil {
					if in.methods[typeName] == nil {
						in.methods[typeName] = make(map[string]*FuncDecl)
					}
					in.methods[typeName][decl.Name.Name] = decl
				}
			} else if decl.Name != nil {
				in.GlobalEnv.Define(decl.Name.Name, Value{Kind: ValFunc, FnVal: decl})
			}
		}
	}

	// Execute top-level statements
	var lastVal Value
	for _, s := range file.Stmts {
		v, err := in.ExecStmt(s, in.GlobalEnv)
		if err != nil {
			return Value{}, in.GlobalEnv.stdout.String(), err
		}
		lastVal = v
	}

	// If main() function exists, call it!
	if mainFn, ok := in.GlobalEnv.Get("main"); ok && mainFn.Kind == ValFunc {
		mainEnv := NewEnv(in.GlobalEnv)
		retVal, err := in.ExecBlock(mainFn.FnVal.Body, mainEnv)
		for i := len(mainEnv.defers) - 1; i >= 0; i-- {
			_, _ = in.Eval(mainEnv.defers[i], mainEnv)
		}
		if err != nil {
			return Value{}, in.GlobalEnv.stdout.String(), err
		}
		return retVal, in.GlobalEnv.stdout.String(), nil
	}

	return lastVal, in.GlobalEnv.stdout.String(), nil
}
