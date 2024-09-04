package models

import (
	_ "github.com/lib/pq" // Импортируем драйвер PostgreSQL
)

type Card struct {
	UserID         int32   `gorm:"not null"`
	CardType       string  `gorm:"not null"`
	CardNumber     string  `gorm:"not null;unique"`
	CardExpiryDate string  `gorm:"not null"`
	Availability   bool    `gorm:"default:true"`
	Username       string  `gorm:"not null"`
	Balance        float64 `gorm:"default:100"`
}

type FintransSuccessfulTransactionsPostgres struct {
	CardNumber          string
	Amount              float64
	RecipientCardNumber string
}
