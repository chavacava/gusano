package lint

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/tools/go/packages"
)

// ReadFile defines an abstraction for reading files.
type ReadFile func(path string) (result []byte, err error)

// Linter is used for linting set of files.
type Linter struct {
	reader ReadFile
}

// New creates a new Linter
func New(reader ReadFile) Linter {
	return Linter{reader: reader}
}

var (
	genHdr = []byte("// Code generated ")
	genFtr = []byte(" DO NOT EDIT.")
)

// Lint lints a set of files with the specified rule.
func (l *Linter) Lint(pkgs []*packages.Package, ruleSet []Rule, config Config) (<-chan Failure, error) {
	failures := make(chan Failure)
	stopFiltering := make(chan struct{})
	unfilteredFailures := make(chan Failure)

	go func() {
		for {
			select {
			case <-stopFiltering:
				return
			case f := <-unfilteredFailures:
				// Implement failure filtering
				failures <- f
			}
		}
	}()

	var wg sync.WaitGroup
	for _, pkg := range pkgs {
		wg.Add(1)
		go func(pkg *packages.Package) {
			if err := l.lintPackage(pkg, ruleSet, config, unfilteredFailures); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			defer wg.Done()
		}(pkg)
	}

	go func() {
		wg.Wait()
		stopFiltering <- struct{}{}
		close(unfilteredFailures)
		close(failures)
	}()

	return failures, nil
}

func (l *Linter) lintPackage(pkg *packages.Package, ruleSet []Rule, config Config, failures chan Failure) error {
	rPkg := &Package{
		fset:      pkg.Fset,
		files:     map[string]*File{},
		Name:      pkg.ID,
		mu:        sync.Mutex{},
		TypesInfo: pkg.TypesInfo,
		TypesPkg:  pkg.Types,
	}

	for _, fileAST := range pkg.Syntax {
		/*
			// TODO detect generated files
			content, err := l.reader(filename)
			if err != nil {
				return err
			}
			if isGenerated(content) && !config.IgnoreGeneratedHeader {
				continue
			}
		*/

		file, err := NewFile(fileAST.Name.Name, rPkg, fileAST)
		if err != nil {
			return err
		}
		rPkg.files[fileAST.Name.Name] = file
	}

	if len(rPkg.files) == 0 {
		return nil
	}

	rPkg.lint(ruleSet, config, failures)

	return nil
}

// isGenerated reports whether the source file is generated code
// according the rules from https://golang.org/s/generatedcode.
// This is inherited from the original go lint.
/*
func isGenerated(src []byte) bool {
	sc := bufio.NewScanner(bytes.NewReader(src))
	for sc.Scan() {
		b := sc.Bytes()
		if bytes.HasPrefix(b, genHdr) && bytes.HasSuffix(b, genFtr) && len(b) >= len(genHdr)+len(genFtr) {
			return true
		}
	}
	return false
}
*/
