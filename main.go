package main

import (
	_ "embed"
	"fmt"
	"os"
	"text/template"

	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"
	"os/exec"
	"path/filepath"
)

var (
	//go:embed template.txt
	tmpl     string
	mainTmpl = template.Must(template.New("main").Parse(tmpl))

	app      = cli.NewApp()
	fuzzFlag = &cli.StringFlag{
		Name:  "func",
		Usage: "The function to fuzz",
		Value: "Fuzz",
	}

	packageFlag = &cli.PathFlag{
		Name:     "package",
		Required: true,
		Usage: `The package-path where the fuzzer resides. OBS! This is not not the same thing as the filesystem path. 

For example, if your fuzzer FuzzBar() resides in  /home/user/go/src/github.com/holiman/bazonk/bar/goo/foo.go, then the 
package-path is 'github.com/holiman/bazonk/bar/goo
'`,
	}

	goPathFlag = &cli.PathFlag{
		Name: "gopath",
		Usage: `If specified, this path is used to search for the repository. If not specified, the $GOPATH env variable 
is used for locating the source code. Note: the source is expected to be found under a '/src' folder beneath the gopath. 

For example, if your fuzzer FuzzBar() resides in  /home/user/go/src/github.com/holiman/bazonk/bar/goo/foo.go, then the 
gopath is '/home/user/go/src'
`,
		Required: false,
		Value:    os.Getenv("GOPATH"),
	}

	outputFlag = &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   "Output-file from compilation",
		Value:   "fuzzer.a",
	}

	buildArgsFlag = &cli.StringSliceFlag{
		Name:  "build.arg",
		Usage: `Arguments passed to the go builder. Example: '--build.arg="-overlay=foo.bar" --build.arg="--race"''`,
	}

	tagsFlag = &cli.StringSliceFlag{
		Name:  "build.tags",
		Usage: `Extra build flags. Example '--build.tags="fo,bar,zoo"'`,
		Value: cli.NewStringSlice("gofuzz_libfuzzer", "libfuzzer"),
	}
)

func init() {
	app.Action = shim
	app.Copyright = "Copyright 2023 Martin Holst Swende"
	app.Flags = []cli.Flag{
		fuzzFlag,
		packageFlag,
		goPathFlag,
		outputFlag,
		buildArgsFlag,
		tagsFlag,
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func shim(ctx *cli.Context) error {

	var (
		targetPkg  = ctx.Path(packageFlag.Name)
		fuzzFunc   = ctx.String(fuzzFlag.Name)
		tags       = ctx.StringSlice(tagsFlag.Name)
		repoRoot   = fmt.Sprintf("%v/src", ctx.String(goPathFlag.Name))
		path       = filepath.Join(repoRoot, targetPkg)
		outputFile = ctx.String(outputFlag.Name)
		buildArgs  = append(ctx.StringSlice(buildArgsFlag.Name), "-gcflags", "all=-d=libfuzzer", "-buildmode=c-archive ")
	)
	slog.Info("Fuzz-builder starting",
		"function", fuzzFunc, "reporoot", repoRoot,
		"package", targetPkg, "abspath", path,
		"output", outputFile, "buildflags", buildArgs,
		"tags", tags)

	ok, err := rewriteTargetFile(path, fuzzFunc, "github.com/holiman/gofuzz-shim/testing")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Nothing to do with target file %v\n", targetPkg)
	}
	main, err := createMain(targetPkg, fuzzFunc)
	if err != nil {
		return err
	}
	return build(main, outputFile, buildArgs, tags)
	/*
		   Executing command: /usr/local/go/bin/go build
		-o bitutil-fuzz.a
		-buildmode c-archive
		-tags gofuzz_libfuzzer,libfuzzer
		-trimpath
		-gcflags
		all=-d=libfuzzer
		# These might be needed? Not certain
		-gcflags syscall=-d=libfuzzer=0 -gcflags runtime/pprof=-d=libfuzzer=0
		./main.3248680275.go
	*/
}

func build(main, out string, buildFlags, tags []string) error {
	args := []string{"build", "-o", out}
	args = append(args, buildFlags...)
	if len(tags) > 0 {
		args = append(args, "-t")
		args = append(args, tags...)
	}
	args = append(args, main)
	cmd := exec.Command("go", args...)
	slog.Info("Building", "command", cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// createMain creates a new main.xx.go-file in the current directory,
// and returns the path to the new file.
func createMain(targetPkg, fuzzFunc string) (string, error) {
	mainFile, err := os.CreateTemp(".", "main.*.go")
	if err != nil {
		slog.Error("Failed to create tempfile", "err", err)
		return "", err
	}
	slog.Info("Wrote main entry point for fuzzing", "file", mainFile.Name())
	defer mainFile.Close()
	type pkgFunc struct {
		PkgPath string
		Func    string
	}
	return mainFile.Name(), mainTmpl.Execute(mainFile, &pkgFunc{targetPkg, fuzzFunc})
}
