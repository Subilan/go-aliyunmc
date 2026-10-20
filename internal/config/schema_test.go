package config

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// 分节路径必须与这份清单一致。
func TestSectionPaths(t *testing.T) {
	out, err := toml.Marshal(Config{})
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var got []string
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			got = append(got, line)
		}
	}

	want := []string{
		"[base]", "[base.cors]", "[base.tls]",
		"[aliyun]", "[aliyun.ecs]", "[aliyun.ecs.system_disk]", "[aliyun.ecs.data_disk]",
		"[server]", "[perms]", "[ssh]",
		"[user_routes]", "[user_routes.chat_token]",
		"[tasks]", "[tasks.deploy]", "[tasks.deploy.vars]",
		"[tasks.backup]", "[tasks.backup.vars]",
		"[tasks.archive]", "[tasks.archive.vars]",
		"[monitors]", "[monitors.server]", "[monitors.instance]", "[monitors.file_sync]",
		"[monitors.auto_archive_idle]", "[monitors.backup]", "[monitors.spot_interruption]",
		"[monitors.best_ecs_candidate]", "[monitors.best_ecs_candidate.filters]",
		"[samplers]", "[samplers.player_count]", "[samplers.balance]",
	}

	slices.Sort(got)
	slices.Sort(want)

	if !slices.Equal(got, want) {
		t.Errorf("分节路径与预期不一致：\n得到: %v\n期望: %v", got, want)
	}
}

// Store 来自另一个文件，不应出现在 config.toml 里。
func TestConfigTOMLHasNoStoreSection(t *testing.T) {
	out, err := toml.Marshal(Config{})
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	if strings.Contains(string(out), "[store]") {
		t.Error("config.toml 不应包含 [store]：数据库配置只存在于 store.toml")
	}
}

// commented 只允许用在叶子字段上：库对结构体字段会把它连同整个表头与子键一起注释掉。
func TestCommentedOnlyOnLeafFields(t *testing.T) {
	checkCommentedTag(t, reflect.TypeFor[Config](), "Config")
}

func checkCommentedTag(t *testing.T, typ reflect.Type, path string) {
	t.Helper()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" || field.Tag.Get("toml") == "-" {
			continue
		}

		fieldPath := path + "." + field.Name
		fieldType := deref(field.Type)

		// 元素是结构体的切片/映射会输出成 [[...]] 或子表，同样算"表"。
		valueKind := fieldType.Kind()
		if valueKind == reflect.Slice || valueKind == reflect.Map {
			valueKind = deref(fieldType.Elem()).Kind()
		}

		commented := slices.Contains(strings.Split(field.Tag.Get("toml"), ","), "commented")
		if commented && valueKind == reflect.Struct {
			t.Errorf("%s 带 commented，但它输出的是表而非叶子", fieldPath)
		}

		switch {
		case fieldType.Kind() == reflect.Struct:
			checkCommentedTag(t, fieldType, fieldPath)
		case (fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Map) && valueKind == reflect.Struct:
			checkCommentedTag(t, deref(fieldType.Elem()), fieldPath+"[]")
		}
	}
}

func deref(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}
