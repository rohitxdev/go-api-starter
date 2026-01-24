package util

import (
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

func Must[T any](fn func() (T, error)) T {
	val, err := fn()
	if err != nil {
		panic(err)
	}
	return val
}
