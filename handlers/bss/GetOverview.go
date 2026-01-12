package bss

import (
	"strconv"
	"time"

	"github.com/Subilan/go-aliyunmc/clients"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/gin-gonic/gin"
)

type Overview struct {
	Balance           float64   `json:"balance"`
	OssExpense        float64   `json:"ossExpense"`
	EcsExpense        float64   `json:"ecsExpense"`
	YunDiskExpense    float64   `json:"yunDiskExpense"`
	CdtExpense        float64   `json:"cdtExpense"`
	TotalExpense      float64   `json:"totalExpense"`
	LatestPayment     float64   `json:"latestPayment"`
	ExpenseDays       int       `json:"expenseDays"`
	ExpenseAverage    float64   `json:"expenseAverage"`
	LatestPaymentTime time.Time `json:"latestPaymentTime"`
}

func HandleGetOverview() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		var result Overview

		queryAccountBalanceResponse, err := clients.BssClient.QueryAccountBalance()

		if err != nil {
			return nil, err
		}

		result.Balance, err = strconv.ParseFloat(*queryAccountBalanceResponse.Body.Data.AvailableAmount, 32)

		if err != nil {
			return nil, err
		}

		err = db.Pool.
			QueryRow("SELECT SUM(amount) FROM transactions WHERE flow='Expense' AND `type`='Consumption' AND remarks = 'OSS'").
			Scan(&result.OssExpense)

		if err != nil {
			return nil, err
		}

		err = db.Pool.
			QueryRow("SELECT SUM(amount) FROM transactions WHERE flow='Expense' AND `type`='Consumption' AND remarks = 'ECS'").
			Scan(&result.EcsExpense)

		if err != nil {
			return nil, err
		}

		err = db.Pool.
			QueryRow("SELECT SUM(amount) FROM transactions WHERE flow='Expense' AND `type`='Consumption' AND remarks = 'CDT_INTERNET_PUBLIC_CN'").
			Scan(&result.CdtExpense)

		if err != nil {
			return nil, err
		}

		err = db.Pool.
			QueryRow("SELECT SUM(amount) FROM transactions WHERE flow='Expense' AND `type`='Consumption' AND remarks = 'YUNDISK'").
			Scan(&result.YunDiskExpense)

		if err != nil {
			return nil, err
		}

		result.TotalExpense = result.OssExpense + result.EcsExpense + result.CdtExpense + result.YunDiskExpense

		err = db.Pool.QueryRow("SELECT AVG(`day_amount`) AS `total_day_averge`, COUNT(*) AS `total_days` FROM (SELECT SUM(`amount`) AS `day_amount`, DATE(`time`) AS `day` FROM transactions WHERE flow='Expense' AND remarks IN ('ECS', 'YUNDISK', 'OSS', 'CDT_INTERNET_PUBLIC_CN') GROUP BY `day`) a").
			Scan(&result.ExpenseAverage, &result.ExpenseDays)

		if err != nil {
			return nil, err
		}

		err = db.Pool.QueryRow("SELECT amount, time FROM transactions WHERE flow='Income' AND `type`='Payment'").Scan(&result.LatestPayment, &result.LatestPaymentTime)

		if err != nil {
			return nil, err
		}

		return helpers.Data(result), nil
	})
}
