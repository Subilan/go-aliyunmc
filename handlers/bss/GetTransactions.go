package bss

import (
	"fmt"
	"time"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

type GetTransactionQuery struct {
	helpers.Paginated
	Remarks string `form:"remarks" binding:"omitempty,oneof=ECS OSS YUNDISK CDT_INTERNET_PUBLIC_CN"`
}

func HandleGetTransactions() gin.HandlerFunc {
	return helpers.QueryHandler[GetTransactionQuery](func(query GetTransactionQuery, c *gin.Context) (any, error) {
		if query.Page == 0 {
			query.Page = 1
		}
		if query.PageSize == 0 {
			query.PageSize = 10
		}

		// Define the date range boundaries
		startBoundary := time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC) // September 2024
		endBoundary := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)   // January 2026

		// Define the base filter conditions to reuse in both queries
		baseFilter := `
			(time < ? OR time >= ?)  -- Exclude records between Sept 2024 and Jan 2026
			AND (
				-- If remarks is empty, only include if flow is "Income"
				(remarks IS NULL OR remarks = '') AND flow = 'Income'
				OR
				-- If remarks is not empty, only include if it's one of the allowed values
				remarks IN ('OSS', 'ECS', 'YUNDISK', 'CDT_INTERNET_PUBLIC_CN')
			)
		`

		// Build the main query with all filtering conditions
		querySQL := fmt.Sprintf(`
			SELECT amount, balance, time, flow, type, remarks, billing_cycle
			FROM transactions
			WHERE %s
		`, baseFilter)

		// Initialize parameters for the base filter
		params := []interface{}{startBoundary, endBoundary}

		// Add the additional filter for the specific remarks value if provided in query
		if query.Remarks != "" {
			querySQL += " AND remarks = ?"
			params = append(params, query.Remarks)
		}

		// Add the ordering and pagination to the main query
		querySQL += " ORDER BY `time` DESC LIMIT ? OFFSET ?"
		params = append(params, query.PageSize, (query.Page-1)*query.PageSize)

		// Execute the main query
		rows, err := db.Pool.Query(querySQL, params...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var transactions []store.Transaction
		for rows.Next() {
			var transaction store.Transaction
			err := rows.Scan(
				&transaction.Amount,
				&transaction.Balance,
				&transaction.Time,
				&transaction.Flow,
				&transaction.Type,
				&transaction.Remarks,
				&transaction.BillingCycle,
			)
			if err != nil {
				return nil, err
			}

			transactions = append(transactions, transaction)
		}

		// Build the count query using the same base filter
		countSQL := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM transactions
			WHERE %s
		`, baseFilter)

		// Initialize count parameters with the same base parameters
		countParams := []interface{}{startBoundary, endBoundary}

		// Add the same additional filter for the specific remarks value if provided
		if query.Remarks != "" {
			countSQL += " AND remarks = ?"
			countParams = append(countParams, query.Remarks)
		}

		var total int
		err = db.Pool.QueryRow(countSQL, countParams...).Scan(&total)
		if err != nil {
			return nil, err
		}

		return helpers.Data(gin.H{
			"data":  transactions,
			"total": total,
		}), nil
	})
}
