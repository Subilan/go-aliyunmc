package store

import (
	"time"
)

// Transaction represents the transactions table structure
type Transaction struct {
	Amount       float64   `json:"amount"`
	Balance      float64   `json:"balance"`
	Time         time.Time `json:"time"`
	Flow         string    `json:"flow"`
	Type         string    `json:"type"`
	Remarks      *string   `json:"remarks,omitempty"`
	BillingCycle string    `json:"billingCycle"`
}
