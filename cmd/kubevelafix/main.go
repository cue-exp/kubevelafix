package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/parser"
	"github.com/cue-exp/kubevelafix"
	"github.com/kr/fs"
)

var errors = 0

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: kubvelafix [directory | cue file]...\n")
		fmt.Fprintf(os.Stderr, `
This command runs the fix on every CUE file mentioned on the command line,
or, for a directory, every CUE file found in that directory.
`)
		os.Exit(2)
	}
	flag.Parse()
	for _, arg := range flag.Args() {
		info, err := os.Stat(arg)
		if err != nil {
			errorf("%v", err)
			continue
		}
		if !info.IsDir() {
			if err := fix(arg); err != nil {
				errorf("%v", err)
			}
			continue
		}
		w := fs.Walk(".")
		for w.Step() {
			if err := w.Err(); err != nil {
				errorf("%v", err)
				continue
			}
			if !w.Stat().Mode().IsRegular() {
				continue
			}
			if filepath.Ext(w.Path()) != ".cue" {
				continue
			}
			if err := fix(w.Path()); err != nil {
				errorf("%v", err)
			}
		}
	}
	if errors > 0 {
		os.Exit(1)
	}
}

func fix(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	f, err := parser.ParseFile(path, data, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("cannot parse %q: %v", path, err)
	}
	f = kubevelafix.Fix(f).(*ast.File)
	data1, err := format.Node(f)
	if err != nil {
		panic(err)
	}
	if bytes.Equal(data, data1) {
		return nil
	}
	if err := os.WriteFile(path, data1, 0o666); err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func errorf(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "kubevelafix: %s\n", fmt.Sprintf(f, a...))
	errors++
}
