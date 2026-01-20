package monitors

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Subilan/go-aliyunmc/clients"
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/filelog"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	bss20171214 "github.com/alibabacloud-go/bssopenapi-20171214/v6/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
)

func fetchAllTransactions(ctx context.Context, client *bss20171214.Client, createTimeStart time.Time) ([]*bss20171214.QueryAccountTransactionDetailsResponseBodyDataAccountTransactionsListAccountTransactionsList, error) {
	var allItems []*bss20171214.QueryAccountTransactionDetailsResponseBodyDataAccountTransactionsListAccountTransactionsList
	var nextToken string
	var total int32

	for {
		req := &bss20171214.QueryAccountTransactionDetailsRequest{
			CreateTimeStart: tea.String(createTimeStart.Format("2006-01-02T15:04:05Z")),
			//CreateTimeEnd:   tea.String(time.Now().Format("2006-01-02T15:04:05Z")),
			NextToken: tea.String(nextToken),
		}

		res, err := client.QueryAccountTransactionDetailsWithContext(ctx, req, &dara.RuntimeOptions{})

		if err != nil {
			return nil, err
		}

		if res.Body.Data == nil {
			return nil, errors.New("body.Data is nil")
		}

		if res.Body.Data.AccountTransactionsList != nil && res.Body.Data.AccountTransactionsList.AccountTransactionsList != nil {
			allItems = append(allItems, res.Body.Data.AccountTransactionsList.AccountTransactionsList...)
			total = *res.Body.Data.TotalCount
			//logger.Println("got length %d, all length %d, total %d, left %d", len(res.Body.Data.AccountTransactionsList.AccountTransactionsList), len(allItems), total, int(total)-len(allItems))

			if total == int32(len(allItems)) || len(res.Body.Data.AccountTransactionsList.AccountTransactionsList) == 0 {
				break
			}
		}

		nextToken = *res.Body.Data.NextToken
	}

	return allItems, nil
}

func BssSync(quit chan bool) {
	logger := filelog.NewLogger("bss-sync", "BssSync")
	logger.Println("starting...")
	cfg := config.Cfg.Monitor.BssSync
	ticker := time.NewTicker(cfg.IntervalDuration())

	for {
		func() {
			logger.Println("begin sync bss info")
			ctx, cancel := context.WithTimeout(context.Background(), cfg.TimeoutDuration())
			defer cancel()
			var latestTransactionTime time.Time

			err := db.Pool.QueryRowContext(ctx, "SELECT `time` FROM transactions ORDER BY `time` DESC LIMIT 1").Scan(&latestTransactionTime)

			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					latestTransactionTime = config.Cfg.Monitor.BssSync.InitialTime
				} else {
					logger.Println("warn: unexpected error during latest transaction time query:", err)
					return
				}
			}

			resp, err := fetchAllTransactions(ctx, clients.BssClient, latestTransactionTime)

			if err != nil {
				logger.Println("warn: cannot fetch transactions: ", err)
				return
			}

			logger.Printf("got %d transactions from api\n", len(resp))

			success := 0

			for _, transaction := range resp {
				parsedTime, err := time.Parse(time.RFC3339, *transaction.TransactionTime)

				if err != nil {
					logger.Printf("warn: transaction time %s is malformed, skipping", *transaction.TransactionTime)
					continue
				}

				result, err := db.Pool.ExecContext(ctx,
					`INSERT INTO transactions (amount, balance, time, flow, type, remarks, billing_cycle) 
VALUES (?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE amount=amount`, *transaction.Amount, *transaction.Balance, parsedTime, *transaction.TransactionFlow, *transaction.TransactionType, *transaction.Remarks, *transaction.BillingCycle)

				if err != nil {
					logger.Println("warn: cannot insert transactions: ", err, "skipping")
				}

				rowsAffected, _ := result.RowsAffected()
				success += int(rowsAffected)
			}

			logger.Println("inserted", success, "transactions")
			logger.Println("next refresh in", cfg.IntervalDuration())
		}()

		select {
		case <-ticker.C:
			continue
		case <-quit:
			return
		}
	}
}
