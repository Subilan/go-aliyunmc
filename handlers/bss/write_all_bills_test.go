package bss

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/helpers/db"
)

func TestWriteAllBills(t *testing.T) {
	config.Load("../../config.toml")
	err := db.InitPool()

	if err != nil {
		t.Error(err)
	}

	for _, productCode := range []string{"ecs", "oss", "yundisk"} {
		marshalled, _ := os.ReadFile("oss_cache_" + productCode + ".json")
		var unmarshalled = make([]map[string]interface{}, 0)

		_ = json.Unmarshal(marshalled, &unmarshalled)

		i := 0

		for _, row := range unmarshalled {
			if row["PaymentAmount"] == 0 {
				continue
			}

			_, err = db.Pool.Exec(
				`INSERT INTO bills (usage_end, usage_start, product_code, product_name, pay, status) 
VALUES (?, ?, ?, ?, ?, ?)`, row["UsageEndTime"], row["UsageStartTime"], row["ProductCode"], row["ProductName"], row["PaymentAmount"], row["Status"])
			if err != nil {
				t.Error(err)
			}
			i++
			t.Logf("%s inserted #%d\n", productCode, i)
		}
	}
}
