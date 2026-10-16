package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"github.com/pelletier/go-toml/v2"

	"github.com/Subilan/go-aliyunmc/log_util"
)

// ConfigDir 是配置目录的默认位置。
const ConfigDir = "configs/"

func configDir() string {
	if dir := os.Getenv("GO_ALIYUNMC_CONFIG_DIR"); dir != "" {
		return dir
	}
	return ConfigDir
}

// MustLoad 读取全部配置：configs/store.toml 与 configs/config.toml。run 使用。
func MustLoad() {
	var cfg Config

	if err := load("store", &cfg.Store); err != nil {
		log_util.Fatal("%v", err)
	}
	if err := load("config", &cfg); err != nil {
		log_util.Fatal("%v", err)
	}

	Set(&cfg)
}

// MustLoadStore 只读取 configs/store.toml，供 migrate 与 create_user 使用。
func MustLoadStore() {
	var cfg Config

	if err := load("store", &cfg.Store); err != nil {
		log_util.Fatal("%v", err)
	}

	Set(&cfg)
}

// load 把一个配置文件读进 obj：严格解码 → 填默认值 → 校验，三个步骤的顺序不可颠倒。
func load(name string, obj any) error {
	path := filepath.Join(configDir(), name+".toml")

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("无法读取配置文件 %s: %w", path, err)
	}

	// DisallowUnknownFields：配置里出现结构体没有的键（拼错、或字段已删除）时直接报错。
	decoder := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
	if err := decoder.Decode(obj); err != nil {
		return fmt.Errorf("无法解析配置文件 %s: %s", path, describeDecodeError(err))
	}

	// defaults.Set 只填零值，配置里显式写过的值不会被覆盖。
	if err := defaults.Set(obj); err != nil {
		return fmt.Errorf("为配置文件 %s 填充默认值失败: %w", path, err)
	}

	if err := validator.New().Struct(obj); err != nil {
		return fmt.Errorf("配置文件 %s 不合要求: %w", path, err)
	}

	return nil
}

// describeDecodeError 返回解码错误的详细信息：StrictMissingError 的逐字段明细在 String() 里。
func describeDecodeError(err error) string {
	var missing *toml.StrictMissingError
	if errors.As(err, &missing) {
		return missing.String()
	}

	return err.Error()
}
