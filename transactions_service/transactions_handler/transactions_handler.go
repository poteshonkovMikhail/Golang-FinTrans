package govno

import (
	"encoding/json"
	"fmt"
	"log"

	bal "fin-trans/transactions_service/balances"

	"github.com/streadway/amqp"
	//"gorm.io/gorm" // Импортируем GORM

	usfl "fin-trans/database_methods_package"
	models "fin-trans/models_package"
	cardpb "fin-trans/proto/proto_generated/cards_service"   // Путь к сгенерированным protobuf-файлам сервиса карт
	pb "fin-trans/proto/proto_generated/transactions_sender" // Путь к сгенерированным protobuf-файлам сервиса транзакций
)

// Функция для чтения данных из очереди RabbitMQ
func ReadFromRabbitMQ(queueName string, cardClient cardpb.CardServiceClient) {
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
		true,      // долговечная
		false,     // автоудаление
		false,     // эксклюзивная
		false,     // без ожидания
		nil,       // аргументы
	)
	if err != nil {
		fmt.Printf("failed to declare a queue: %v", err)
	}

	if err := usfl.Db.Ping(); err != nil {
		log.Println("Транзакция висит в очереди и ждёт установления соединения с БД, запросы пользователей продолжают поступать в очередь")
		//go bal.RefreshBalances(models.FintransSuccessfulTransactionsPostgres{}, cardClient)
	} else {

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

		go func() {
			// Потоковое чтение всех сообщений из очереди
			for msg := range msgs {
				err := json.Unmarshal(msg.Body, &transaction)
				if err != nil {
					fmt.Printf("failed to unmarshal message: %v", err)
					continue
				}
				log.Printf("Received: %s", transaction)

				newTransaction := models.FintransSuccessfulTransactionsPostgres{
					CardNumber:          transaction.CardNumber,
					Amount:              transaction.Amount,
					RecipientCardNumber: transaction.RecipientCardNumber,
				}
				go bal.RefreshBalances(newTransaction, cardClient)

			}
		}()

	}
}
