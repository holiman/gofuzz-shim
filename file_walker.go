// This file comes from https://github.com/AdamKorcz/go-118-fuzz-build
// Copyright @AdamKorcz
// Modifications copyright Martin Holst Swende 2023
package main

import (
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"os"

	"golang.org/x/exp/slog"
	"golang.org/x/tools/go/ast/astutil"
)

func rewriteImport(path, fuzzerName, newImport string) (restoreFn func(), err error) {
	var fset = token.NewFileSet()
	astFile, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	// Replace import path, if needed
	if astutil.DeleteImport(fset, astFile, "testing") {
		astutil.AddImport(fset, astFile, newImport)
	} else {
		slog.Warn("No imports to replace", "file", path)
	}
	// Write into new file
	fuzzPath := path + "_fuzz.go"
	if newFile, err := os.Create(fuzzPath); err != nil {
		return nil, fmt.Errorf("failed to create new file: %v", err)
	} else {
		printer.Fprint(newFile, fset, astFile)
		newFile.Close()
		slog.Info("Created new file", "name", newFile.Name())
	}
	// Rename old file
	savePath := fmt.Sprintf("%v.orig", path)
	slog.Info("Saving original file", "path", savePath)
	if err := os.Rename(path, savePath); err != nil {
		return nil, err
	}
	restoreFunc := func() {
		slog.Info("Restoring files", "restoring", path, "removing", fuzzPath)
		os.Remove(fuzzPath)
		os.Rename(savePath, path)
	}
	return restoreFunc, nil
}
