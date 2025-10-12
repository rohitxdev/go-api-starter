package util

import "github.com/go-playground/validator"

func Coalesce[T comparable](values ...T) T {
	var zero T
	for _, v := range values {
		if v != zero {
			return v
		}
	}
	return zero
}

var Validate = validator.New()
