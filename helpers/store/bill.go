package store

import (
	"time"
)

// Bill represents the bills table structure
type Bill struct {
	UsageEnd    time.Time `json:"usageEnd"`
	UsageStart  time.Time `json:"usageStart"`
	ProductCode string    `json:"productCode"`
	ProductName string    `json:"productName"`
	Pay         float64   `json:"pay"`
	Status      string    `json:"status"`
}
