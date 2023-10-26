package faketest

import (
	"reflect"

	"github.com/holiman/gofuzz-shim/input"
)

type F struct {
	common
	data []byte
}

func NewF(data []byte) *F {
	return &F{data: data}
}

// Add will add the arguments to the seed corpus for the fuzz test. This will be
// a no-op if called after or within the fuzz target, and args must match the
// arguments for the fuzz target.
// NOT implemented
func (f *F) Add(args ...any) {}

func (f *F) Fuzz(ff any) {
	input.NewSource(f.data).FillAndCall(ff, reflect.ValueOf(new(T)))
}
