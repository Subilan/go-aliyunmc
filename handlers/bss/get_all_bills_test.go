package bss

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Subilan/go-aliyunmc/clients"
	"github.com/Subilan/go-aliyunmc/config"
	bss20171214 "github.com/alibabacloud-go/bssopenapi-20171214/v6/client"
	"github.com/alibabacloud-go/tea/tea"
)

func fetchAllBills(client *bss20171214.Client, billingCycle, productCode string, t *testing.T) ([]*bss20171214.QueryBillResponseBodyDataItemsItem, error) {
	var allItems []*bss20171214.QueryBillResponseBodyDataItemsItem
	pageNum := int32(1)
	var total int32

	for {
		req := &bss20171214.QueryBillRequest{
			BillingCycle:     tea.String(billingCycle),
			ProductCode:      tea.String(productCode),
			PageNum:          tea.Int32(pageNum),
			PageSize:         tea.Int32(300),
			IsHideZeroCharge: tea.Bool(true),
		}

		res, err := client.QueryBill(req)

		if err != nil {
			t.Fatal(err.Error())
		}

		if res.Body.Data != nil && res.Body.Data.Items != nil && res.Body.Data.Items.Item != nil {
			allItems = append(allItems, res.Body.Data.Items.Item...)
			total = *res.Body.Data.TotalCount

			if total == int32(len(allItems)) || len(res.Body.Data.Items.Item) == 0 {
				break
			}
		}

		pageNum++
	}

	return allItems, nil
}

func TestRetrieveAndSaveBills(t *testing.T) {
	start := time.Date(2021, 2, 0, 0, 0, 0, 0, time.UTC)
	end := time.Now()

	var dates = make([]string, 0, 12*5)

	for {
		if start.After(end) {
			break
		}
		dates = append(dates, start.Format("2006-01"))
		start = start.AddDate(0, 1, 0)
	}

	t.Log(dates)

	var productCodes = []string{"oss", "ecs", "yundisk"}

	config.Load("../../config.toml")
	client, _ := clients.ShouldCreateBssClient()

	for _, productCode := range productCodes {
		var allBills = make([]*bss20171214.QueryBillResponseBodyDataItemsItem, 0, 1000)
		for _, date := range dates {
			t.Logf("productCode: %s, date: %s", productCode, date)
			bills, err := fetchAllBills(client, date, productCode, t)

			if err != nil {
				t.Fatal(err.Error())
			}

			t.Log("length:", len(bills))
			allBills = append(allBills, bills...)
		}

		marshalled, _ := json.Marshal(allBills)

		err := os.WriteFile("oss_cache_"+productCode+".json", marshalled, 0600)

		if err != nil {
			t.Fatal(err.Error())
		}

		t.Log("save bills successfully")
	}
}
