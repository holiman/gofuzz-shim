package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func copyFile(t *testing.T, src, dst string) error {
	t.Helper()
	srcData, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, srcData, 0666)
}

func nicediff(have, want []byte) string {
	var i = 0
	for ; i < len(have) && i < len(want); i++ {
		if want[i] != have[i] {
			break
		}
	}
	var end = i + 40
	var start = i - 50
	if start < 0 {
		start = 0
	}
	var h, w string
	if end < len(have) {
		h = string(have[start:end])
	} else {
		h = string(have[start:])
	}
	if end < len(want) {
		w = string(want[start:end])
	} else {
		w = string(want[start:])
	}
	return fmt.Sprintf("have vs want:\n%q\n%q\n", h, w)
}

func compareFiles(t *testing.T, havePath, wantPath string) {
	var have, want []byte
	var err error
	t.Helper()
	if have, err = os.ReadFile(havePath); err != nil {
		t.Fatal(err)
	}
	if want, err = os.ReadFile(wantPath); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(have, want) {
		t.Log(nicediff(have, want))
		t.Error("have != want")
	}
}

// TestRewrites rewrites the import of a test-file.
func TestRewrite(t *testing.T) {
	d := t.TempDir()
	if err := copyFile(t, "./testdata/target/target1_test.go.txt", filepath.Join(d, "target1_test.go")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(d, "target1_test.go")
	_, err := rewriteImport(path, "FuzzEncoder", "github.com/baz/bazonk")
	if err != nil {
		t.Fatal(err)
	}
	compareFiles(t, filepath.Join(d, "target1_test.go_fuzz.go"), "./testdata/target/target1_test.go.rewritten.txt")
}

func TestGenerateMain(t *testing.T) {
	f, err := createMain("github.com/ethereum/go-ethereum/common/bitutil", "FuzzEncoder")
	if err != nil {
		t.Fatal(err)
	}
	compareFiles(t, f, "./testdata/main.output.want")

	t.Cleanup(func() { os.Remove(f) })

}
