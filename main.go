/*
 * Copyright 2018. bigpigeon. All rights reserved.
 * Use of this source code is governed by a MIT style
 * license that can be found in the LICENSE file.
 */

package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\ttoy-doctor [directory]\n")
	fmt.Fprintf(os.Stderr, "\ttoy-doctor files... # Must be a single package\n")
	flag.PrintDefaults()
}

func Main(args []string) {
	if len(args) == 0 {
		// Default: process whole package in current directory.
		args = []string{"."}
	}

	// Parse the package once.
	var (
		dir   string
		files []*ast.File
		walk  *Walker
		err   error
	)
	fs := token.NewFileSet()

	if len(args) == 1 && isDirectory(args[0]) {
		dir = args[0]
		pkgMap, err := parser.ParseDir(fs, dir, nil, 0)
		if err != nil {
			panic(err)
		}
		for _, pkg := range pkgMap {
			for _, f := range pkg.Files {
				files = append(files, f)
			}
		}
	} else {
		dir = filepath.Dir(args[0])
		for _, arg := range args {
			f, err := parser.ParseFile(fs, arg, nil, 0)
			if err != nil {
				panic(err)
			}
			files = append(files, f)
		}
	}
	walk, err = NewWalker(fs, dir, files)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		ast.Walk(walk, file)
	}
	fmt.Println(walk.Report())
}

func isDirectory(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		panic(err)
	}
	return info.IsDir()
}

func main() {
	flag.Usage = Usage
	flag.Parse()
	args := flag.Args()
	Main(args)
}
