package mc

import (
	"os"
	"testing"

	"github.com/Subilan/go-aliyunmc/internal/testutil"
)

func TestMain(m *testing.M) {
	if err := testutil.ChdirProjectRoot(); err != nil {
		panic(err)
	}
	if err := LoadData(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestAdvancements(t *testing.T) {
	t.Logf("advancements: %d", len(Advancements()))
}
