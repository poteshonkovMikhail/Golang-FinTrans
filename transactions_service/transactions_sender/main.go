package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	usfl "fin-trans/database_methods_package"
	cardpb "fin-trans/proto/proto_generated/cards_service" // Путь к сгенерированным protobuf-файлам сервиса карт
	rds "fin-trans/proto/proto_generated/redis_cache_service"
	pb "fin-trans/proto/proto_generated/transactions_sender" // Путь к сгенерированным protobuf-файлам сервиса транзакций (этого сервиса)
	trhr "fin-trans/transactions_service/transactions_handler"
)

var (
	QueueName = "TransactionsQueue"
)

type server struct {
	pb.UnimplementedTransactionServiceServer
	cardClient  cardpb.CardServiceClient
	rabbitConn  *amqp.Connection
	redisServer *redis.Client
	redisClient rds.CardServiceClient
}

func (s *server) SendTransactionToQueue(ctx context.Context, req *pb.CreateTransactionRequest) {
	//Вынесено в отдельную горутину для асинхронного выполнения
	go func() {
		transaction := map[string]interface{}{
			"card_number":           req.CardNumber,
			"amount":                req.Amount,
			"recipient_card_number": req.RecipientCardNumber,
		}

		// Открываем канал для сообщений RabbitMQ
		channel, err := s.rabbitConn.Channel()
		if err != nil {
			log.Printf("Ошибка при открытии канала для сообщений: %v", err)
			return
		}
		defer channel.Close()

		body, err := json.Marshal(transaction)
		if err != nil {
			log.Printf("Ошибка при json энкодинге транзакции: %v", err)
		}

		err = channel.Publish(
			"",        // Прямой обменник
			QueueName, // Имя очереди
			false,     // Признак сходимости
			false,     // Признак приоритета
			amqp.Publishing{
				ContentType:  "application/json",
				Body:         body,
				DeliveryMode: amqp.Persistent, // сообщение будет устойчивым
			})
		if err != nil {
			log.Printf("Ошибка при отправке транзакции в обменник сообщений: %v", err)
		}
	}()
}

func (s *server) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.CreateTransactionResponse, error) {
	//Здесь сначала бежим проверять наличие карты в репликации части основных данных в Redis
	cardRes, err := s.redisClient.RedisGetCard(ctx, &rds.RedisGetCardRequest{CardNumber: req.CardNumber})
	if err != nil || cardRes == nil {
		//Если в Redis не найдена искомая карта
		cardRes, err := s.cardClient.GetCard(ctx, &cardpb.GetCardRequest{CardNumber: req.CardNumber})
		// Кэшируем результат
		go func(cardRes *cardpb.GetCardResponse) {

			//Здесь сделать наличия выделенного места в памяти

			jsonCard, _ := json.Marshal(cardRes)
			s.redisServer.Set(ctx, cardRes.CardNumber, jsonCard, 48*time.Hour)
		}(cardRes)

		if err != nil {
			log.Printf("Не найдена карта при создании транзакции: %v", err)

			//Запуск горутины, отправляющей
			go s.SendTransactionToQueue(ctx, req)
			return &pb.CreateTransactionResponse{
				IsCreated: false,
				Message:   "Перевод успешно начат, вы получите уведомление, когда транзакция завершится",
			}, nil
		}
	}
	if cardRes.CardNumber == "" {
		return &pb.CreateTransactionResponse{
			IsCreated: false,
			Message:   "У вас нет такой карты",
		}, nil
	}

	if req.Amount <= 0.0 {
		return &pb.CreateTransactionResponse{
			IsCreated: false,
			Message:   "Введите корректное значение в поле Amount",
		}, nil
	}

	if cardRes.Balance >= req.Amount {

		//Запуск горутины, отправляющей
		go s.SendTransactionToQueue(ctx, req)

		return &pb.CreateTransactionResponse{
			IsCreated: true,
			Message:   "Перевод успешно начат, вы получите уведомление, когда транзакция завершится",
		}, nil
	} else {
		return &pb.CreateTransactionResponse{
			IsCreated: false,
			Message:   "Недостаточно средств, пополните баланс или поробуйте другую карту",
		}, nil
	}
}

func newServer(cardClient cardpb.CardServiceClient, rabbitConn *amqp.Connection, rdb *redis.Client, redisClient rds.CardServiceClient) *server {
	return &server{
		cardClient:  cardClient,
		rabbitConn:  rabbitConn,
		redisServer: rdb,
		redisClient: redisClient,
	}
}

func startGRPCServer() {
	// Настройка подключения к gRPC серверу CardService
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("не удалось подключиться к CardService: %v", err)
	}
	defer conn.Close()
	cardClient := cardpb.NewCardServiceClient(conn)
	if cardClient == nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}

	//Запускаем чтение сообщений из RabbitMQ
	go trhr.ReadFromRabbitMQ(QueueName, cardClient)

	// Настройка подключения к RabbitMQ
	rabbitConn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("не удалось подключиться к RabbitMQ: %v", err)
	}
	defer rabbitConn.Close()

	//Подключение к БД
	// Initialize your database connection details
	connPostgres := &usfl.ConnPostgres{
		Host:     "localhost",
		User:     "postgres",
		Password: "workout+5",
		DbName:   "fintrans_transactions_postgres",
		Port:     "5432",
		SslMode:  "disable",
	}

	// Call the DbConnector method
	if err := connPostgres.DbConnector(usfl.Db_transactions_sevice_conn); err != nil {
		fmt.Println("Error connecting to the database:", err)
	} else {
		fmt.Println("Successfully connected to the database!")
	}

	//Подключаемся к gRPC сервису RedisCacheServer
	connline, err := grpc.Dial("localhost:50053", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("не удалось подключиться к CardService: %v", err)
	}
	defer conn.Close()
	redisClient := rds.NewCardServiceClient(connline)
	if redisClient == nil {
		log.Printf("Ошибка при запуске кэш-сервера Redis: %v", err)
	}
	/////////////////////////////////////
	//Подключаемся к Redis
	// Создаем новый клиент Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // адрес Redis-сервера
		Password: "",               // пароль (если установлен)
		DB:       0,                // используемая база данных
	})

	ctx := context.Background()

	// Проверяем подключение
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("Ошибка при подключении к Redis: %v", err)
	}
	fmt.Println("Успешно подключено к Redis!")

	// Запись данных в Redis
	//err = rdb.Set(ctx, "key", "value", 10*time.Second).Err()
	//if err != nil {
	//	log.Fatalf("Ошибка при записи в Redis: %v", err)
	//}
	//fmt.Println("Данные записаны в Redis.")

	// Чтение данных из Redis
	//val, err := rdb.Get(ctx, "key").Result()
	//if err != nil {
	//	log.Fatalf("Ошибка при чтении из Redis: %v", err)
	//}
	//fmt.Printf("Значение по ключу 'key': %s\n", val)

	// Закрываем соединение с Redis
	//defer rdb.Close()
	///////////////////////

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterTransactionServiceServer(grpcServer, newServer(cardClient, rabbitConn, rdb, redisClient))
	reflection.Register(grpcServer)

	log.Println("Transactions gRPC server is running on port: 50052")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

}

func startRESTServer() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterTransactionServiceHandlerFromEndpoint(ctx, mux, "localhost:50052", opts)
	if err != nil {
		log.Fatalf("Failed to register gRPC gateway: %v", err)
	}

	// Запускаем HTTP сервер

	log.Println("HTTP сервер запущен на порту: 8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Не удалось запустить HTTP сервер: %v", err)
	}

}

func main() {
	go startGRPCServer()
	startRESTServer()
}
