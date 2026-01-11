package bss

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/helpers/db"
)

func TestWriteAllTransactions(t *testing.T) {
	config.Load("../../config.toml")
	err := db.InitPool()

	if err != nil {
		t.Error(err)
	}
	marshalled, _ := os.ReadFile("transactions.json")
	var unmarshalled = make([]map[string]interface{}, 0)

	_ = json.Unmarshal(marshalled, &unmarshalled)

	i := 0

	for _, row := range unmarshalled {
		if row["Amount"] == 0 {
			continue
		}

		parsedTime, err := time.Parse(time.RFC3339, row["TransactionTime"].(string))

		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Pool.Exec(
			`INSERT INTO transactions (amount, balance, time, flow, type, remarks, billing_cycle) 
VALUES (?, ?, ?, ?, ?, ?, ?)`, row["Amount"], row["Balance"], parsedTime, row["TransactionFlow"], row["TransactionType"], row["Remarks"], row["BillingCycle"])
		if err != nil {
			t.Fatal(err)
		}
		i++
		t.Logf("inserted #%d\n", i)
	}
}
