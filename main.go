package main

import (
	_ "embed"
	"fmt"
	"os"
	"text/template"

	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"
	"os/exec"
	"strings"
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

	targetsFlag = &cli.StringSliceFlag{
		Name:    "fiximports",
		Aliases: []string{"f"},
		Usage: `Target file(s) to rewrite imports of. This is typically: 
  1. The ".._test.go"-file which contains the main 'Fuzz(testing.F)'-function, and 
  2. Any other ".._test.go"-files which (1) relies upon, e.g. common testing-utilities or types.
`,
		Value: cli.NewStringSlice("gofuzz_libfuzzer", "libfuzzer"),
	}

	packageFlag = &cli.PathFlag{
		Name:     "package",
		Required: true,
		Usage: `The package-path where the fuzzer resides. OBS! This is not not the same thing as the filesystem path. 

For example, if your fuzzer FuzzBar() resides in  /home/user/go/src/github.com/holiman/bazonk/bar/goo/foo.go, then the 
package-path is 'github.com/holiman/bazonk/bar/goo
'`,
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
		Name:    "build.tags",
		Aliases: []string{"tags"},
		Usage:   `Extra build flags. Example '--build.tags="fo,bar,zoo"'`,
		Value:   cli.NewStringSlice("gofuzz_libfuzzer", "libfuzzer"),
	}
)

func init() {
	app.Action = shim
	app.Copyright = "Copyright 2023 Martin Holst Swende"
	app.Flags = []cli.Flag{
		fuzzFlag,
		targetsFlag,
		packageFlag,
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
		targetPkg   = ctx.Path(packageFlag.Name)
		targetFiles = ctx.StringSlice(targetsFlag.Name)
		fuzzFunc    = ctx.String(fuzzFlag.Name)
		tags        = ctx.StringSlice(tagsFlag.Name)
		outputFile  = ctx.String(outputFlag.Name)
		buildArgs   = append(ctx.StringSlice(buildArgsFlag.Name), "-gcflags", "all=-d=libfuzzer", "-buildmode=c-archive")
	)
	slog.Info("Fuzz-builder starting",
		"function", fuzzFunc, "to-rewrite", strings.Join(targetFiles, ","),
		"package", targetPkg, "output", outputFile, "buildflags", buildArgs,
		"tags", tags)
	for _, path := range targetFiles {
		slog.Info("Rewriting imports", "file", path)
		restoreFn, err := rewriteImport(path, fuzzFunc, "github.com/holiman/gofuzz-shim/testing")
		if err != nil {
			return err
		}
		defer restoreFn()
	}
	main, err := createMain(targetPkg, fuzzFunc)
	if err != nil {
		return err
	}
	if err := goTidy(); err != nil {
		return err
	}
	return build(main, outputFile, buildArgs, tags)
}

func build(main, out string, buildFlags, tags []string) error {
	args := []string{"build", "-o", out}
	args = append(args, buildFlags...)
	if len(tags) > 0 {
		args = append(args, "-tags", strings.Join(tags, ","))
	}
	args = append(args, main)
	cmd := exec.Command("go", args...)
	slog.Info("Building", "command", cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, string(out))
		return err
	}
	return nil
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

func goTidy() error {
	if out, err := exec.Command("go", "mod", "tidy").CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, string(out))
		return err
	}
	return nil
}
