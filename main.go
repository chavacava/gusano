package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join([]string(*i), " ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var excludePkgs arrayFlags
	var excludeFiles arrayFlags

	flag.Var(&excludePkgs, "ep", "regex of packages to exclude")
	flag.Var(&excludeFiles, "ef", "regex of files to exclude")
	flag.Parse()

	// Many tools pass their command-line arguments (after any flags)
	// uninterpreted to packages.Load so that it can interpret them
	// according to the conventions of the underlying build system.
	cfg := &packages.Config{Mode: packages.LoadAllSyntax}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load: %v\n", err)
		//os.Exit(1)
	}
	if packages.PrintErrors(pkgs) > 0 {
		//os.Exit(1)
	}

	unused := &Unused{}
	// Print the names of the source files
	// for each package listed on the command line.
	for _, pkg := range pkgs {
		unused.Apply(pkg)
	}
}
