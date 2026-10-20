package mc

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if err := LoadData(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestAdvancements(t *testing.T) {
	t.Logf("advancements: %d", len(Advancements()))
}
