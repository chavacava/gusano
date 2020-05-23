package rule

import (
	"fmt"
	"go/ast"
	"sync"

	"github.com/chavacava/gusano/lint"
)

type toIgnoreType map[*lint.Package]map[*ast.Ident]bool

// UnusedSymbolRule lints unused params in functions.
type UnusedSymbolRule struct {
	sync.Mutex
	toIgnore toIgnoreType
}

func (r *UnusedSymbolRule) createToIgnore(pkg *lint.Package) {
	if r.toIgnore == nil {
		r.toIgnore = toIgnoreType{}
	}
	if r.toIgnore[pkg] == nil {
		r.toIgnore[pkg] = map[*ast.Ident]bool{}
	}

	return
}

func (r *UnusedSymbolRule) ignore(pkg *lint.Package, id *ast.Ident) {
	r.Lock()
	defer r.Unlock()

	r.createToIgnore(pkg)
	_, ok := r.toIgnore[pkg][id]
	if ok {
		panic("symbol defined twice " + id.Name)
	}

	r.toIgnore[pkg][id] = true
}

// ApplyToPackage applies the rule to given package.
func (r *UnusedSymbolRule) ApplyToPackage(pkg *lint.Package, arguments lint.Arguments, failures chan lint.Failure) {
	r.Lock()
	defer r.Unlock()

	if pkg.TypesInfo == nil {
		return
	}

	for id, d := range pkg.TypesInfo.Defs {
		isInitFunc := id.String() == "init" // TODO provide more precise init func identification
		isMainFunc := id.String() == "main" && pkg.IsMain()
		mustIgnore := d == nil || isInitFunc || isMainFunc || id.IsExported() || id.String() == "_" || r.toIgnore[pkg][id]
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
				kind = r.retrieveIdKind(id.Obj.Decl, id.Obj.Kind.String())
			}

			//			fmt.Printf("unused %v (%+v)\n", id, id.Obj)
			failures <- lint.Failure{
				Confidence: 1,
				Failure:    fmt.Sprintf("unused %v %v", kind, d.Name()),
				Node:       id,
				Position:   lint.FailurePosition{Start: pkg.Fset().Position(id.Pos())},
			}
		}
	}

	delete(r.toIgnore, pkg)
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
	ss := &symbolScanner{r, file.Pkg}
	ast.Walk(ss, file.AST)

	return nil
}

// Name returns the rule name.
func (r *UnusedSymbolRule) Name() string {
	return "unused-symbol"
}

type symbolScanner struct {
	r   *UnusedSymbolRule
	pkg *lint.Package
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

func (s symbolScanner) ignoreAllIdUnder(node ast.Node) {
	ignorer := &ignorer{&s}
	ast.Walk(ignorer, node)
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
	ss *symbolScanner
}

func (w ignorer) Visit(node ast.Node) ast.Visitor {
	switch v := node.(type) {
	case *ast.Ident:
		//		fmt.Printf("ignoring %v %+v\n", v, v.Obj)
		w.ss.r.ignore(w.ss.pkg, v)
	}

	return w
}
