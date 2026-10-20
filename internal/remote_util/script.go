package remote_util

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"text/template"
)

// RenderScriptTemplate 渲染 fsys 中 name 指向的脚本模板，结果保证以换行结尾
func RenderScriptTemplate(fsys fs.FS, name string, vars any) (string, error) {
	content, err := fs.ReadFile(fsys, name)
	if err != nil {
		return "", fmt.Errorf("读取模板文件失败(%s): %w", name, err)
	}

	tmpl, err := template.New(path.Base(name)).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("解析模板失败(%s): %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("渲染模板失败(%s): %w", name, err)
	}

	script := buf.String()
	if !strings.HasSuffix(script, "\n") {
		script += "\n"
	}

	return script, nil
}
