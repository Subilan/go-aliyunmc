package config

import (
	"reflect"

	"github.com/go-playground/validator/v10"
)

func validPositiveRange(fl validator.FieldLevel) bool {
	v := fl.Field()

	if v.Kind() != reflect.Slice {
		return false
	}

	if v.Len() != 2 {
		return false
	}

	a := v.Index(0).Int()
	b := v.Index(1).Int()

	return a > 0 && b > 0 && a <= b
}

type IntRange struct {
	Min int
	Max int
}
