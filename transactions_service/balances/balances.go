package balances

import (
	"context"
	"log"

	usfl "fin-trans/database_methods_package"
	models "fin-trans/models_package"
	cardpb "fin-trans/proto/proto_generated/cards_service"
)

func RefreshBalances(newTransaction models.FintransSuccessfulTransactionsPostgres, cardClient cardpb.CardServiceClient) {
	ctx := context.Background()
	// Начинаем новую транзакцию
	tx, err := usfl.Db_transactions_sevice_conn.Begin()
	if err != nil {
		log.Printf("Ошибка при начале транзакции: %v", err)
		return
	}
	/////////////////////////////////////////////////////////////////Если ложиться нахой сервис карт, то направляем все это в очередь//////////////////////////////////////////////////
	/////////////////////////////////////////////////////////////////Добавить каналы для отправки ответов пользователю///////

	// Получаем информацию о картах отправителя и получателя
	senderCard, err := cardClient.GetCard(ctx, &cardpb.GetCardRequest{CardNumber: newTransaction.CardNumber})
	if err != nil {
		tx.Rollback()
		log.Printf("Ошибка при получении карты отправителя: %v", err)
		return
	}

	recipientCard, err := cardClient.GetCard(ctx, &cardpb.GetCardRequest{CardNumber: newTransaction.RecipientCardNumber})
	if err != nil {
		tx.Rollback()
		log.Printf("Ошибка при получении карты получателя: %v", err)
		return
	}

	if senderCard != nil {
		// Проверяем, достаточно ли средств для отправки
		if senderCard.Balance < newTransaction.Amount {
			tx.Rollback()
			log.Printf("Недостаточно средств, пополните баланс или воспользуйтесь другой картой: %v", err)
			return
		}
	}
	if recipientCard != nil {
		// Проверяем доступность получателя
		if !recipientCard.Availability {
			tx.Rollback()
			log.Printf("Откат транзакции: карта недоступна %v", newTransaction.RecipientCardNumber)
			return
		}
	}

	// Обновляем балансы пользователей
	if _, err := tx.Exec("UPDATE cards SET balance = balance - $1 WHERE card_number = $2", newTransaction.Amount, senderCard.CardNumber); err != nil {
		tx.Rollback()
		log.Println("Отмена транзакции (ошибка при обновлении баланса отправителя)")
		return
	}

	if _, err := tx.Exec("UPDATE cards SET balance = balance + $1 WHERE card_number = $2", newTransaction.Amount, recipientCard.CardNumber); err != nil {
		tx.Rollback()
		log.Printf("Ошибка при обновлении баланса получателя: %v", err)
		return
	}

	// Сохраняем информацию о транзакции в БД
	_, err = tx.Exec("INSERT INTO fintrans_successful_transactions_postgres (card_number, recipient_card_number, amount) VALUES ($1, $2, $3)",
		newTransaction.CardNumber, newTransaction.RecipientCardNumber, newTransaction.Amount)
	if err != nil {
		tx.Rollback()
		log.Printf("Ошибка при сохранении транзакции в БД: %v", err)
		return
	}

	// Подтверждаем транзакцию
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		log.Printf("Ошибка при подтверждении транзакции: %v", err)
	}

	log.Println("Транзакция прошла успешно")
}
