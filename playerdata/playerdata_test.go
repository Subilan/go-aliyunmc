package playerdata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/testutil"
)

func TestMain(m *testing.M) {
	if err := testutil.ChdirProjectRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "chdir to project root failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func writeStatsFile(t *testing.T, uuid string, playTime int) string {
	t.Helper()

	if err := os.MkdirAll(StatsDir, 0o755); err != nil {
		t.Fatalf("create stats dir failed: %v", err)
	}

	path := filepath.Join(StatsDir, uuid+".json")
	content := fmt.Sprintf(`{"stats":{"minecraft:custom":{"minecraft:play_time":%d}},"DataVersion":1}`, playTime)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write stats file failed: %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })

	return path
}

func TestReadStatsCachesResult(t *testing.T) {
	uuid := fmt.Sprintf("test-cache-%d", time.Now().UnixNano())
	path := writeStatsFile(t, uuid, 100)

	first, err := ReadStats(uuid)
	if err != nil {
		t.Fatalf("first read failed: %v", err)
	}

	if err := os.WriteFile(path, []byte(`{"stats":{"minecraft:custom":{"minecraft:play_time":200}},"DataVersion":1}`), 0o644); err != nil {
		t.Fatalf("overwrite stats file failed: %v", err)
	}

	second, err := ReadStats(uuid)
	if err != nil {
		t.Fatalf("second read failed: %v", err)
	}

	if string(first) != string(second) {
		t.Fatalf("expected the cached stats, got %q then %q", string(first), string(second))
	}

	var parsed struct {
		Custom map[string]float64 `json:"minecraft:custom"`
	}
	if err := json.Unmarshal(second, &parsed); err != nil {
		t.Fatalf("unmarshal stats failed: %v", err)
	}
	if parsed.Custom["minecraft:play_time"] != 100 {
		t.Fatalf("expected play_time 100 from the cache, got %v", parsed.Custom["minecraft:play_time"])
	}
}

func TestReadStatsDoesNotCacheMissingFile(t *testing.T) {
	uuid := fmt.Sprintf("test-missing-%d", time.Now().UnixNano())

	if _, err := ReadStats(uuid); !os.IsNotExist(err) {
		t.Fatalf("expected a not-exist error, got %v", err)
	}

	writeStatsFile(t, uuid, 100)

	stats, err := ReadStats(uuid)
	if err != nil {
		t.Fatalf("read after creation failed: %v", err)
	}
	if len(stats) == 0 {
		t.Fatal("expected stats to be readable after the file appears")
	}
}

func TestListPlayerUUIDsCachesResult(t *testing.T) {
	if err := os.MkdirAll(StatsDir, 0o755); err != nil {
		t.Fatalf("create stats dir failed: %v", err)
	}

	first, err := ListPlayerUUIDs()
	if err != nil {
		t.Fatalf("first listing failed: %v", err)
	}

	uuid := fmt.Sprintf("test-listing-%d", time.Now().UnixNano())
	path := writeStatsFile(t, uuid, 100)

	second, err := ListPlayerUUIDs()
	if err != nil {
		t.Fatalf("second listing failed: %v", err)
	}

	for _, listed := range second {
		if listed == uuid {
			t.Fatalf("expected the cached listing to exclude %s", uuid)
		}
	}
	if len(first) != len(second) {
		t.Fatalf("expected the cached listing to keep %d entries, got %d", len(first), len(second))
	}

	os.Remove(path)
}
