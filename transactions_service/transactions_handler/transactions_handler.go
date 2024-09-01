package govno

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	//"time"

	bal "fin-trans/transactions_service/balances"

	"github.com/streadway/amqp"

	cardpb "fin-trans/transactions_service/proto/proto_generated/cards_service"   // Путь к сгенерированным protobuf-файлам сервиса карт
	pb "fin-trans/transactions_service/proto/proto_generated/transactions_sender" // Путь к сгенерированным protobuf-файлам сервиса транзакций
)

// Функция для чтения данных из очереди RabbitMQ
func ReadFromRabbitMQ(queueName string, db *sql.DB, cardClient cardpb.CardServiceClient) {
	var transaction *pb.CreateTransactionRequest

	Conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	channel, err := Conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	// Объявление очереди
	q, err := channel.QueueDeclare(
		queueName, // имя очереди
		false,     // долговечная
		false,     // автоудаление
		false,     // эксклюзивная
		false,     // без ожидания
		nil,       // аргументы
	)
	if err != nil {
		fmt.Printf("failed to declare a queue: %v", err)
	}

	// Получение сообщений из очереди
	msgs, err := channel.Consume(
		q.Name, // имя очереди
		"",     // потребитель
		true,   // автоподтверждение
		false,  // эксклюзивная
		false,  // без ожидания
		false,  // без аргументов
		nil,
	)
	if err != nil {
		fmt.Printf("failed to register a consumer: %v", err)
	}

	//Потоковое чтение всех сообщений из очереди
	for msg := range msgs {
		err := json.Unmarshal(msg.Body, &transaction)
		if err != nil {
			fmt.Printf("failed to unmarshal message: %v", err)
		}
		log.Printf("Received: %s", transaction)
		// Начало транзакции PostgreSQL
		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}
		// Добавление записи в базу данных
		_, err = tx.Exec("INSERT INTO fintrans_successful_transactions_postgres (card_number, amount, recipient_card_number) VALUES ($1, $2, $3)", transaction.CardNumber, transaction.Amount, transaction.RecipientCardNumber)
		if err != nil {
			tx.Rollback() // Откат транзакции в случае ошибки
			log.Fatal("Transaction rolled back due to error:", err)
		}
		go checkFinallyTransactionsTerms(transaction.RecipientCardNumber, transaction.CardNumber, transaction.Amount, tx, cardClient, db)
	}

}

func checkFinallyTransactionsTerms(recipientCardNumber string, senderCardNumber string, amount float64, tx *sql.Tx, cardClient cardpb.CardServiceClient, db *sql.DB) {

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	go func() {

		recipientCard, err := cardClient.CheckRecipientCard(ctx, &cardpb.CheckRecipientCardRequest{RecipientCardNumber: recipientCardNumber})
		cancel()
		if err != nil {
			log.Printf("Ошибка проверки карты: %v", err)
			tx.Rollback() // Откат при ошибке
			return
		}

		if !recipientCard.Availability {
			tx.Rollback()
			log.Printf("Откат транзакции: карта недоступна %v", recipientCardNumber)
			return
		}
		go bal.RefreshBalances(tx, recipientCardNumber, senderCardNumber, amount, cardClient, db)
	}()
}
