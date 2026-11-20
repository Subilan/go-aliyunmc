package labmodels

import (
	"slices"
	"testing"
)

func TestSplitPlayerNames(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "空字符串", raw: "", want: nil},
		{name: "单个玩家", raw: "Alice", want: []string{"Alice"}},
		{name: "多个玩家", raw: "Alice,Bob,张三", want: []string{"Alice", "Bob", "张三"}},
		{name: "空白项", raw: " Alice, ,Bob ,, ", want: []string{"Alice", "Bob"}},
		{name: "只有空白", raw: " , , ", want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitPlayerNames(tt.raw)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("SplitPlayerNames(%q) = %#v, want %#v", tt.raw, got, tt.want)
			}
		})
	}
}
