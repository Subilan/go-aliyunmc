package config

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
)

func expectValidationError(err error, t *testing.T) {
	if err == nil {
		t.Fatal("expected error, got none")
	}

	var validationErrors validator.ValidationErrors

	if !errors.As(err, &validationErrors) {
		t.Fatalf("unexpected error %T, %s", err, err)
	}

	t.Logf("got expected error %v", err.Error())
}

func TestOK(t *testing.T) {
	err := load("../testdata/test_config.good.toml")

	if err != nil {
		t.Error(err)
	}

	t.Cleanup(func() {
		Cfg = Config{}
	})
}

func TestMissingRequiredField(t *testing.T) {
	err := load("../testdata/test_config.bad.required.toml")

	expectValidationError(err, t)

	t.Cleanup(func() {
		Cfg = Config{}
	})
}

func TestInvalidJavaVersion(t *testing.T) {
	err := load("../testdata/test_config.bad.javaVersion.toml")

	expectValidationError(err, t)

	t.Cleanup(func() {
		Cfg = Config{}
	})
}
