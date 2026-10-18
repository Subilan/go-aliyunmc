// Package policygen 根据策略配置渲染 RAM 自定义策略模板。
//
// 模板与 example 配置纳入版本控制；真实配置（账号、地域、bucket 等）位于
// policies/policy.toml，被 .gitignore 忽略。
package policygen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/pelletier/go-toml/v2"
)

const (
	// DefaultConfigPath 是真实策略配置的默认路径。
	DefaultConfigPath = "policies/policy.toml"
	// DefaultTemplatePath 是策略模板的默认路径。
	DefaultTemplatePath = "policies/templates/policy.json.tmpl"
	// ExampleConfigPath 是纳入版本控制的示例配置路径。
	ExampleConfigPath = "policies/policy.example.toml"
)

// Config 是策略配置文件的结构。
type Config struct {
	Common       Common                 `toml:"common"`
	Environments map[string]Environment `toml:"environments"`
}

// Common 是所有环境共享的字段。
type Common struct {
	// AccountID 是阿里云账号 ID（主账号），同一账号下的 RAM 子用户共用。
	AccountID string `toml:"account_id"`
}

// Environment 是一个环境的策略参数。
type Environment struct {
	Name      string `toml:"name"`
	RegionID  string `toml:"region_id"`
	OSSBucket string `toml:"oss_bucket"`
	OSSPrefix string `toml:"oss_prefix"`
}

// templateData 是传给模板的数据，字段名与模板中的占位符对应。
type templateData struct {
	Environment string
	Name        string
	AccountID   string
	RegionID    string
	OSSBucket   string
	OSSPrefix   string
}

var (
	accountIDRe = regexp.MustCompile(`^[0-9]+$`)
	regionRe    = regexp.MustCompile(`^[a-z0-9-]+$`)
	bucketRe    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	nameRe      = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	prefixRe    = regexp.MustCompile(`^[A-Za-z0-9._/-]*$`)
)

// Load 读取并校验策略配置。未知键会直接报错。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	decoder := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("解析策略配置 %s 失败: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("策略配置 %s 无效: %w", path, err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	c.Common.AccountID = strings.TrimSpace(c.Common.AccountID)
	if !accountIDRe.MatchString(c.Common.AccountID) {
		return fmt.Errorf("common.account_id 必须是非空数字")
	}

	if len(c.Environments) == 0 {
		return fmt.Errorf("至少需要配置一个 [environments.<name>]")
	}

	for key, env := range c.Environments {
		env.Name = strings.TrimSpace(env.Name)
		env.RegionID = strings.TrimSpace(env.RegionID)
		env.OSSBucket = strings.TrimSpace(env.OSSBucket)
		env.OSSPrefix = normalizePrefix(env.OSSPrefix)

		switch {
		case strings.TrimSpace(key) == "":
			return fmt.Errorf("环境名称不能为空")
		case !nameRe.MatchString(env.Name):
			return fmt.Errorf("environments.%s.name 只能包含字母、数字、下划线和连字符", key)
		case !regionRe.MatchString(env.RegionID):
			return fmt.Errorf("environments.%s.region_id 格式不正确", key)
		case !bucketRe.MatchString(env.OSSBucket):
			return fmt.Errorf("environments.%s.oss_bucket 格式不正确", key)
		case !prefixRe.MatchString(env.OSSPrefix):
			return fmt.Errorf("environments.%s.oss_prefix 格式不正确", key)
		}

		c.Environments[key] = env
	}

	return nil
}

// normalizePrefix 把前缀规范成 "dev/" 形式；空字符串表示整个 bucket。
func normalizePrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix
}

// Render 渲染策略模板。envFilter 为空时输出全部环境，否则只输出指定环境。
//
// 输出全部环境时，每份策略前带两行 # 注释标明环境名与策略名；
// 指定 envFilter 时只输出纯 JSON，便于直接复制。
func Render(cfg *Config, templatePath, envFilter string) (string, error) {
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("读取策略模板失败: %w", err)
	}

	tmpl, err := template.New("policy").Option("missingkey=error").Parse(string(templateBytes))
	if err != nil {
		return "", fmt.Errorf("解析策略模板失败: %w", err)
	}

	names := make([]string, 0, len(cfg.Environments))
	for name := range cfg.Environments {
		names = append(names, name)
	}
	sort.Strings(names)

	if envFilter != "" {
		if _, ok := cfg.Environments[envFilter]; !ok {
			return "", fmt.Errorf("未知环境 %q；可选环境: %s", envFilter, strings.Join(names, ", "))
		}
		names = []string{envFilter}
	}

	var out strings.Builder
	for i, name := range names {
		env := cfg.Environments[name]
		data := templateData{
			Environment: name,
			Name:        env.Name,
			AccountID:   cfg.Common.AccountID,
			RegionID:    env.RegionID,
			OSSBucket:   env.OSSBucket,
			OSSPrefix:   env.OSSPrefix,
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("渲染环境 %s 失败: %w", name, err)
		}

		rendered := strings.TrimSpace(buf.String())
		if !json.Valid([]byte(rendered)) {
			return "", fmt.Errorf("环境 %s 的渲染结果不是合法 JSON", name)
		}

		if envFilter != "" {
			return rendered + "\n", nil
		}

		if i > 0 {
			out.WriteString("\n")
		}
		fmt.Fprintf(&out, "# environment: %s\n# policy: %s\n%s\n", name, env.Name, rendered)
	}

	return out.String(), nil
}
