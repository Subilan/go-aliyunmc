package tasks

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

type TaskOutput struct {
	Step   uint   `json:"step"`
	Output string `json:"output"`
}

var validate = validator.New()

// ShouldBindArgs 尝试将args绑定到结构体指针out上，并调用validator对结构体结果进行验证。
func ShouldBindArgs[T any](args map[string]any, out *T) error {
	if out == nil {
		return errors.New("out must not be nil")
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           out,
		TagName:          "json", // 使用json tag规定的字段名进行绑定
		ErrorUnused:      true,
		WeaklyTypedInput: false,
		ZeroFields:       true, // 写入结构体之前将结构体清空为零值
	})
	if err != nil {
		return fmt.Errorf("init decoder failed: %w", err)
	}

	if err := decoder.Decode(args); err != nil {
		return fmt.Errorf("decode args failed: %w", err)
	}

	if err := validate.Struct(out); err != nil {
		return fmt.Errorf("validate args failed: %w", err)
	}

	return nil
}
