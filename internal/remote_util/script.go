package remote_util

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// RenderScriptTemplate 从指定路径读取脚本模板文件，并使用提供的变量进行渲染，返回渲染后的脚本内容
func RenderScriptTemplate(templatePath string, vars any) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("读取模板文件失败(%s): %w", templatePath, err)
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("解析模板失败(%s): %w", templatePath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("渲染模板失败(%s): %w", templatePath, err)
	}

	script := buf.String()
	if !strings.HasSuffix(script, "\n") {
		script += "\n"
	}

	return script, nil
}
