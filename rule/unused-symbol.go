package rule

import (
	"fmt"
	"go/ast"
	"sync"

	"github.com/chavacava/gusano/lint"
)

// UnusedSymbolRule lints unused params in functions.
type UnusedSymbolRule struct {
	sync.Mutex
	toIgnore map[*ast.Ident]bool
}

func (r *UnusedSymbolRule) createToIgnore() {
	if r.toIgnore != nil {
		return
	}
	r.toIgnore = map[*ast.Ident]bool{}
}

func (r *UnusedSymbolRule) ignore(idNode *ast.Ident) {
	r.Lock()
	defer r.Unlock()
	_, ok := r.toIgnore[idNode]
	if ok {
		panic("symbol defined twice " + idNode.Name)
	}
	r.createToIgnore()
	r.toIgnore[idNode] = true
}

// ApplyToPackage applies the rule to given package.
func (r *UnusedSymbolRule) ApplyToPackage(pkg *lint.Package, arguments lint.Arguments, failures chan lint.Failure) {
	r.Lock()
	defer r.Unlock()
	for id, d := range pkg.TypesInfo.Defs {
		isInitFunc := id.String() == "init" // TODO provide more precise init func identification
		mustIgnore := d == nil || isInitFunc || id.IsExported() || id.String() == "_" || r.toIgnore[id]
		if mustIgnore {
			continue
		}

		found := false
		for _, u := range pkg.TypesInfo.Uses {
			if u == d {
				found = true
				break
			}
		}

		if !found {
			kind := "method"

			if id.Obj != nil {
				//fmt.Printf("%v\n%v\n---\n", id.Obj.Decl, id.Obj.Kind.String())
				kind = r.retrieveIdKind(id.Obj.Decl, id.Obj.Kind.String())
			}

			failures <- lint.Failure{
				Confidence: 1,
				Failure:    fmt.Sprintf("unused %v %v", kind, d.Name()),
				Node:       id,
				Position:   lint.FailurePosition{Start: pkg.Fset().Position(id.Pos())},
			}
		}
	}
}
func (r *UnusedSymbolRule) retrieveIdKind(t interface{}, defaultValue string) string {
	if defaultValue == "" {
		defaultValue = "method"
	}
	switch t.(type) {
	case *ast.Field:
		return "field"
	case *ast.ValueSpec:
		// can not determine if it is a const or a var
		return defaultValue
	case *ast.FuncDecl:
		return "function"
	case *ast.TypeSpec:
		return "type"
	default:
		return defaultValue
	}
}

// ApplyToFile applies the rule to given file.
func (r *UnusedSymbolRule) ApplyToFile(file *lint.File, arguments lint.Arguments) []lint.Failure {
	ss := &symbolScanner{r}
	ast.Walk(ss, file.AST)

	return nil
}

// Name returns the rule name.
func (r *UnusedSymbolRule) Name() string {
	return "unused-symbol"
}

type symbolScanner struct {
	r *UnusedSymbolRule
}

func (s symbolScanner) ignoreAllIdUnder(node ast.Node) {
	ignorer := &ignorer{s.r}
	ast.Walk(ignorer, node)
}

func (w symbolScanner) Visit(node ast.Node) ast.Visitor {
	switch v := node.(type) {
	case *ast.Field:
		if len(v.Names) == 0 { // embedded type
			w.ignoreAllIdUnder(v.Type)
		}
	case *ast.InterfaceType:
		if v.Methods != nil {
			w.ignoreAllIdUnder(v.Methods)
		}
		return nil
	case *ast.FuncLit:
		w.ignoreFuncType(v.Type)
		return nil
	case *ast.FuncType:
		w.ignoreFuncType(v)
		return nil
	case *ast.FuncDecl:
		if v.Recv != nil {
			w.ignoreAllIdUnder(v.Recv)
		}

		w.ignoreFuncType(v.Type)
		return nil
	}

	return w
}

func (w symbolScanner) ignoreFuncType(ft *ast.FuncType) {
	if ft == nil {
		return
	}

	if ft.Params != nil {
		w.ignoreAllIdUnder(ft.Params)
	}
	if ft.Results != nil {
		w.ignoreAllIdUnder(ft.Results)
	}
}

type ignorer struct {
	r *UnusedSymbolRule
}

func (w ignorer) Visit(node ast.Node) ast.Visitor {
	switch v := node.(type) {
	case *ast.Ident:
		w.r.ignore(v)
	}

	return w
}
