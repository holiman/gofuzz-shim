package faketest

import (
	"time"
)

type T struct {
	common
}

func NewT() *T {
	return &T{}
}

func (t *T) Deadline() (time.Time, bool)        { return time.Time{}, false }
func (t *T) Run(name string, f func(t *T)) bool { panic("not implemented") }
func (t *T) Parallel()                          {}
func (t *T) Setenv(key, value string)           {}
