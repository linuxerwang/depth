package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/KyleBanks/depth"
)

const (
	outputPadding    = "  "
	outputPrefix     = "├ "
	outputPrefixLast = "└ "
)

var outputJSON bool

type summary struct {
	numInternal int
	numExternal int
	numTesting  int
}

func main() {
	t, pkgs := parse(os.Args[1:])
	if err := handlePkgs(t, pkgs, outputJSON); err != nil {
		os.Exit(1)
	}
}

// parse constructs a depth.Tree from command-line arguments, and returns the
// remaining user-supplied package names
func parse(args []string) (*depth.Tree, []string) {
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	var t depth.Tree
	f.BoolVar(&t.ResolveInternal, "internal", false, "If set, resolves dependencies of internal (stdlib) packages.")
	f.BoolVar(&t.ResolveTest, "test", false, "If set, resolves dependencies used for testing.")
	f.IntVar(&t.MaxDepth, "max", 0, "Sets the maximum depth of dependencies to resolve.")
	f.BoolVar(&outputJSON, "json", false, "If set, outputs the depencies in JSON format.")
	f.Parse(args)

	return &t, f.Args()
}

// handlePkgs takes a slice of package names, resolves a Tree on them,
// and outputs each Tree to Stdout.
func handlePkgs(t *depth.Tree, pkgs []string, outputJSON bool) error {
	for _, pkg := range pkgs {

		err := t.Resolve(pkg)
		if err != nil {
			fmt.Printf("'%v': FATAL: %v\n", pkg, err)
			return err
		}

		if outputJSON {
			writePkgJSON(os.Stdout, *t.Root)
			continue
		}

		writePkg(os.Stdout, *t.Root, 0, false)
		writePkgSummary(os.Stdout, *t.Root)
	}
	return nil
}

// writePkgSummary writes a summary of all packages in a tree
func writePkgSummary(w io.Writer, pkg depth.Pkg) {
	var sum summary
	set := make(map[string]struct{})
	for _, p := range pkg.Deps {
		collectSummary(&sum, p, set)
	}
	out := fmt.Sprintf("%d dependencies (%d internal, %d external, %d testing).",
	                    sum.numInternal + sum.numExternal,
											sum.numInternal,
											sum.numExternal,
										  sum.numTesting)
	w.Write([]byte(out))
}

func collectSummary(sum *summary, pkg depth.Pkg, nameSet map[string]struct{}) {
	if _, ok := nameSet[pkg.Name]; !ok {
		nameSet[pkg.Name] = struct{}{}
		if pkg.Internal {
			sum.numInternal++
		} else {
			sum.numExternal++
		}
		if pkg.Test {
			sum.numTesting++
		}
		for _, p := range pkg.Deps {
			collectSummary(sum, p, nameSet)
		}
	}
}

// writePkgJSON writes the full Pkg as JSON to the provided Writer.
func writePkgJSON(w io.Writer, p depth.Pkg) {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	e.Encode(p)
}

// writePkg recursively prints a Pkg and its dependencies to the Writer provided.
func writePkg(w io.Writer, p depth.Pkg, indent int, isLast bool) {
	var prefix string
	if indent > 0 {
		prefix = outputPrefix

		if isLast {
			prefix = outputPrefixLast
		}
	}

	out := fmt.Sprintf("%v%v%v\n", strings.Repeat(outputPadding, indent), prefix, p.String())
	w.Write([]byte(out))

	for idx, d := range p.Deps {
		writePkg(w, d, indent+1, idx == len(p.Deps)-1)
	}
}
