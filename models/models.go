// models/models.go
package models

import "time"

type Transaction struct {
	ID          int
	Amount      float64
	Description string
	DateTime    time.Time
	IsPositive  bool
	Category    string
	Tags        []string
	Currency    string
	Notes       string
}

type FinanceData struct {
	Transactions []Transaction
	Balances     map[string]float64
}

type WorkEntry struct {
	Date      string // Формат: "2006-01-02"
	Place     string
	StartTime string // Формат: "15:04"
	EndTime   string // Формат: "15:04"
	IsDayOff  bool
}

type WorkLogData struct {
	Entries []WorkEntry
}
