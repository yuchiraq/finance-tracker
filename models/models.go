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
	Balances     map[string]float64 // Баланс для каждой валюты
}
