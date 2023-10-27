// This file comes from https://github.com/AdamKorcz/go-118-fuzz-build
// Copyright @AdamKorcz
// Modifications copyright Martin Holst Swende 2023
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"

	"golang.org/x/exp/slog"
	"golang.org/x/tools/go/ast/astutil"
	"path/filepath"
	"strings"
)

func rewriteTargetFile(path, fuzzerName, newImport string) (ok bool, err error) {
	// Find which file to operate on
	files, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	for _, fd := range files {
		if fd.IsDir() || !strings.HasSuffix(fd.Name(), "_test.go") {
			continue
		}
		if done, err := tryRewriteTargetFile(filepath.Join(path, fd.Name()), fuzzerName, newImport); err != nil {
			return false, err
		} else if done {
			return true, nil
		}
	}
	return false, nil
}

func tryRewriteTargetFile(path, fuzzerName, newImport string) (ok bool, err error) {
	var fset = token.NewFileSet()
	astFile, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return false, err
	}
	if present, err := containsMethod(astFile, fuzzerName); err != nil || !present {
		return false, err
	}
	// Replace import path, if needed
	if !astutil.DeleteImport(fset, astFile, "testing") {
		// Maybe the user is trying to re-run it after already succeding once. If so, just continue
		if astutil.UsesImport(fset, newImport) {
			slog.Info("File already instrumented", "file", path)
			return true, nil
		}
		return false, nil // nothing to do here
	}
	astutil.AddImport(fset, astFile, newImport)
	// Write into new file
	if newFile, err := os.Create(path + "_fuzz.go"); err != nil {
		return false, fmt.Errorf("failed to create new file: %v", err)
	} else {
		printer.Fprint(newFile, fset, astFile)
		newFile.Close()
		slog.Info("Created new file", "name", newFile.Name())
	}
	// Rename old file
	slog.Info("Saving original file", "path", fmt.Sprintf("%v.orig", path))
	return true, os.Rename(path, fmt.Sprintf("%v.orig", path))
}

// containsMethod parses the file at path, and returns true if it contains
// a go function with the given name
func containsMethod(astFile *ast.File, name string) (bool, error) {
	for _, decl := range astFile.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Name.Name == name {
				return true, nil
			}
		}
	}
	return false, nil
}
