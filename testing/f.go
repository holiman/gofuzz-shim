package testing

import (
	"reflect"

	"github.com/holiman/gofuzz-shim/input"
)

type F struct {
	common
	s *input.Source
}

func NewF(data []byte) *F {
	return &F{s: input.NewSource(data)}
}

// Add will add the arguments to the seed corpus for the fuzz test. This will be
// a no-op if called after or within the fuzz target, and args must match the
// arguments for the fuzz target.
// NOT implemented
func (f *F) Add(args ...any) {}

func (f *F) Fuzz(ff any) {
	f.s.FillAndCall(ff, reflect.ValueOf(new(T)))
}

// ReturnValue returns a value for libfuzzer. Docs:
//
// > By default, the fuzzing engine will generate input of any arbitrary length.
// > This might be useful to try corner cases that could lead to a security
// > vulnerability. However, if large inputs are not necessary to increase the
// > coverage of your target API, it is important to add a limit here to
// > significantly improve performance.
// >
// > 	if (size < kMinInputLength || size > kMaxInputLength)
// > 		return 0;
//
// By default: return 1
// We do this by checking how much data the fuzzer tried to consume.
func (f *F) ReturnValue() int {
	if f.s.IsExhausted() {
		return 0
	}
	// We're a bit lenient, but if the input is >2x the used portion, then return
	// 0 to limit it. (== used amount < remaining amount)
	if f.s.Used() < f.s.Len() {
		return 0
	}
	return 1
}
