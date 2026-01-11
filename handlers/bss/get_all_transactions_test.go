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

func fetchAllTransactions(client *bss20171214.Client, t *testing.T) ([]*bss20171214.QueryAccountTransactionDetailsResponseBodyDataAccountTransactionsListAccountTransactionsList, error) {
	var allItems []*bss20171214.QueryAccountTransactionDetailsResponseBodyDataAccountTransactionsListAccountTransactionsList
	var nextToken string
	var total int32

	for {
		req := &bss20171214.QueryAccountTransactionDetailsRequest{
			CreateTimeStart: tea.String("2021-02-01T00:00:00Z"),
			CreateTimeEnd:   tea.String(time.Now().Format("2006-01-02T15:04:05Z")),
			NextToken:       tea.String(nextToken),
		}

		res, err := client.QueryAccountTransactionDetails(req)

		if err != nil {
			t.Fatal(err.Error())
		}

		if res.Body.Data == nil {
			t.Fatal("Body.Data is empty")
		}

		if res.Body.Data.AccountTransactionsList != nil && res.Body.Data.AccountTransactionsList.AccountTransactionsList != nil {
			allItems = append(allItems, res.Body.Data.AccountTransactionsList.AccountTransactionsList...)
			total = *res.Body.Data.TotalCount
			t.Logf("got length %d, all length %d, total %d, left %d", len(res.Body.Data.AccountTransactionsList.AccountTransactionsList), len(allItems), total, int(total)-len(allItems))

			if total == int32(len(allItems)) || len(res.Body.Data.AccountTransactionsList.AccountTransactionsList) == 0 {
				break
			}
		}

		nextToken = *res.Body.Data.NextToken
	}

	return allItems, nil
}

func TestGetAllTransactions(t *testing.T) {
	config.Load("../../config.toml")
	client, _ := clients.ShouldCreateBssClient()

	allItems, err := fetchAllTransactions(client, t)

	if err != nil {
		t.Fatal(err.Error())
	}

	marshalled, _ := json.Marshal(allItems)

	err = os.WriteFile("transactions.json", marshalled, 0644)
}
