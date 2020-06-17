package main

import (
	"fmt"
	"go/ast"
	"sync"

	"golang.org/x/tools/go/packages"
)

type toIgnoreType map[*packages.Package]map[*ast.Ident]bool

func (ig toIgnoreType) contains(pkg *packages.Package, id *ast.Ident) bool {
	return ig[pkg][id]
}

// Unused lints unused params in functions.
type Unused struct {
	sync.Mutex
	toIgnore toIgnoreType
}

func (r *Unused) createToIgnore(pkg *packages.Package) {
	if r.toIgnore == nil {
		r.toIgnore = toIgnoreType{}
	}
	if r.toIgnore[pkg] == nil {
		r.toIgnore[pkg] = map[*ast.Ident]bool{}
	}
}

func (r *Unused) ignore(pkg *packages.Package, id *ast.Ident) {
	r.Lock()
	defer r.Unlock()

	r.createToIgnore(pkg)
	_, ok := r.toIgnore[pkg][id]
	if ok {
		panic("symbol defined twice " + id.Name)
	}

	r.toIgnore[pkg][id] = true
}

// Apply applies the rule to given package.
func (r *Unused) Apply(pkg *packages.Package) {
	for _, f := range pkg.Syntax {
		r.applyToFile(f, pkg)
	}

	r.Lock()
	defer r.Unlock()

	if pkg.TypesInfo == nil {
		return
	}

	for id, d := range pkg.TypesInfo.Defs {
		isInitFunc := id.String() == "init" // TODO && id.Obj != nil && id.Obj.Kind == ast.Fun
		isMainFunc := id.String() == "main" // && id.Obj != nil && id.Obj.Kind == ast.Fun && pkg.Module.Main
		mustIgnore := d == nil || id.IsExported() || id.String() == "_" || isInitFunc || isMainFunc
		if mustIgnore || r.toIgnore.contains(pkg, id) {
			continue
		}

		found := false
		for _, u := range pkg.TypesInfo.Uses {
			if u == d {
				found = true
				break
			}
		}

		if found {
			continue
		}

		position := pkg.Fset.Position(id.NamePos)

		fmt.Printf("%s: %s unused\n", position.String(), id.Name)
	}

	delete(r.toIgnore, pkg)
}

// ApplyToFile applies the rule to given file.
func (r *Unused) applyToFile(file *ast.File, pkg *packages.Package) {
	ss := &symbolScanner{r, pkg}
	ast.Walk(ss, file)
}

type symbolScanner struct {
	r   *Unused
	pkg *packages.Package
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

func (w symbolScanner) ignoreAllIdUnder(node ast.Node) {
	ignorer := &ignorer{&w}
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
		w.ss.r.ignore(w.ss.pkg, v)
	}

	return w
}
