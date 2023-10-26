package faketest

import (
	"fmt"
	"os"
)

type common struct {
	cleanups []func()
}

func (c *common) Cleanup(fn func()) {
	c.cleanups = append(c.cleanups, fn)
}

// Finished triggers the execution of cleanup-functions.
func (c *common) Finished() {
	for i := len(c.cleanups) - 1; i > 0; i-- {
		c.cleanups[i]()
	}
}

func (c *common) Log(args ...any)                 { fmt.Print(args...) }
func (c *common) Logf(format string, args ...any) { fmt.Printf(format, args...) }
func (c *common) Name() string                    { return "libFuzzer" }

// TempDir returns a temporary directory for the test to use.
// The directory is automatically removed by Cleanup when the test and all its
// subtests complete. Each subsequent call to t.TempDir returns a unique directory
func (c *common) TempDir() string {
	dir, err := os.MkdirTemp("", "fuzzdir-")
	if err != nil {
		panic(err)
	}
	c.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func (c *common) Skip(args ...any)                 {}
func (c *common) SkipNow()                         {}
func (c *common) Skipf(format string, args ...any) {}
func (c *common) Skipped() bool                    { return false }
func (c *common) Helper()                          {}
func (c *common) Failed() bool                     { return false }

func (c *common) Error(args ...any)         { panic(fmt.Sprint(args...)) }
func (c *common) Errorf(f string, a ...any) { panic(fmt.Sprintf(f, a...)) }
func (c *common) Fail()                     { panic("Fail()") }
func (c *common) FailNow()                  { panic("FailNow()") }
func (c *common) Fatal(args ...any)         { panic(fmt.Sprint(args...)) }
func (c *common) Fatalf(f string, a ...any) { panic(fmt.Sprintf(f, a...)) }
