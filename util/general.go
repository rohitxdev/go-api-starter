package util

import (
	"runtime/debug"

	"github.com/go-playground/validator"
)

var Validate = validator.New()

func Coalesce[T comparable](values ...T) T {
	var zero T
	for _, v := range values {
		if v != zero {
			return v
		}
	}
	return zero
}

func CapturePanic[T any](fn func() T) (res T, panicVal any, stack []byte) {
	defer func() {
		if r := recover(); r != nil {
			panicVal = r
			stack = debug.Stack()
		}
	}()
	res = fn()
	return
}

func Must[T any](fn func() (T, error)) T {
	val, err := fn()
	if err != nil {
		panic(err)
	}
	return val
}
