package monitors

import "testing"

func TestParseSystemMetrics(t *testing.T) {
	output := `cpu_total=1000
cpu_idle=400
mem_total_kb=8388608
mem_available_kb=6291456
cpu_cores=4
uptime_seconds=3600
`

	raw, err := parseSystemMetrics(output)
	if err != nil {
		t.Fatalf("parseSystemMetrics() error = %v", err)
	}

	if raw.cpuTotal != 1000 || raw.cpuIdle != 400 {
		t.Fatalf("CPU 字段解析不正确: %+v", raw)
	}
	if raw.memTotalKB != 8388608 || raw.memAvailableKB != 6291456 {
		t.Fatalf("内存字段解析不正确: %+v", raw)
	}
	if raw.cpuCores != 4 || raw.uptimeSeconds != 3600 {
		t.Fatalf("核心数或运行时长解析不正确: %+v", raw)
	}
}

func TestParseSystemMetricsRejectsIncompleteOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{name: "缺少字段", output: "cpu_total=1\ncpu_idle=1\n"},
		{name: "值无效", output: "cpu_total=abc\ncpu_idle=1\nmem_total_kb=1\nmem_available_kb=1\ncpu_cores=1\nuptime_seconds=1\n"},
		{name: "内存总量为零", output: "cpu_total=1\ncpu_idle=1\nmem_total_kb=0\nmem_available_kb=0\ncpu_cores=1\nuptime_seconds=1\n"},
		{name: "核心数为零", output: "cpu_total=1\ncpu_idle=1\nmem_total_kb=1\nmem_available_kb=1\ncpu_cores=0\nuptime_seconds=1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseSystemMetrics(tt.output); err == nil {
				t.Fatal("应当返回错误")
			}
		})
	}
}

func TestMemoryUsedBytes(t *testing.T) {
	if got := memoryUsedBytes(100, 40); got != 60*1024 {
		t.Fatalf("memoryUsedBytes(100, 40) = %d, want %d", got, 60*1024)
	}
	if got := memoryUsedBytes(100, 100); got != 0 {
		t.Fatalf("memoryUsedBytes(100, 100) = %d, want 0", got)
	}
	if got := memoryUsedBytes(100, 120); got != 0 {
		t.Fatalf("memoryUsedBytes(100, 120) = %d, want 0", got)
	}
}
