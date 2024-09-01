package balances

import (
	"context"
	"database/sql"
	"log"

	cardpb "fin-trans/transactions_service/proto/proto_generated/cards_service"
)

func RefreshBalances(tx *sql.Tx, recipientCardNumber string, senderCardNumber string, amount float64, cardClient cardpb.CardServiceClient, db *sql.DB) {

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	senderCard, err := cardClient.GetCard(ctx, &cardpb.GetCardRequest{CardNumber: senderCardNumber})
	if err != nil {
		log.Fatalf("Ошибка при получении карты отправителя в методе RefreshBalances: %v", err)
	}

	recipientCard, err := cardClient.GetCard(ctx, &cardpb.GetCardRequest{CardNumber: recipientCardNumber})
	cancel()
	if err != nil {
		log.Fatalf("Ошибка при получении карты получателя в методе RefreshBalances: %v", err)
	}

	// Записываем новые балансы в базу данных
	_, err = tx.Exec("UPDATE user_balances SET balance = $1 WHERE user_id = $2", senderCard.Balance-amount, senderCard.UserId)
	if err != nil {
		log.Fatalf("Ошибка при обновлении баланса отправителя: %v", err)
	}
	_, err = tx.Exec("UPDATE user_balances SET balance = $1 WHERE user_id = $2", recipientCard.Balance+amount, recipientCard.UserId)
	if err != nil {
		log.Fatalf("Ошибка при обновлении баланса получателя: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Ошибка при подтверждении транзакции: %v", err)
	}
	log.Println("Transaction succsess")
}
