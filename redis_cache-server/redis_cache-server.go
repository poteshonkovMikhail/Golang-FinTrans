package main

import (
	"context"
	"encoding/json"
	"net"
	"strconv"

	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
	//"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	usfl "fin-trans/database_methods_package"
	cardpb "fin-trans/proto/proto_generated/cards_service" // Путь к сгенерированным protobuf-файлам сервиса карт
	rds "fin-trans/proto/proto_generated/redis_cache_service"
)

var (
	redisClient *redis.Client
)

type server struct {
	rds.UnimplementedCardServiceServer
	cardClient cardpb.CardServiceClient
	rdb        *redis.Client
}

type UserCard struct {
	ID             int    `json:"id"`
	UserID         int    `json:"user_id"`
	CardNumber     string `json:"card_number"`
	ExpirationDate string `json:"expiration_date"`
}

func startGRPCserver() {
	// Устанавливаем соединение с Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

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

	//Здесь метод в котором реплицируем часть БД в Redis
	go MakeRedisReplicate()

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	rds.RegisterCardServiceServer(grpcServer, newServer(cardClient))
	reflection.Register(grpcServer)

	log.Println("gRPC Redis-Cache-Server is running on port: 50053")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

}

func newServer(cardClient cardpb.CardServiceClient) *server {
	return &server{
		cardClient: cardClient,
	}
}

func (s *server) RedisGetCard(ctx context.Context, req *rds.RedisGetCardRequest) (*rds.RedisGetCardResponse, error) {

	// Пробуем получить данные из кэша
	cachedCardNumber, err := s.rdb.Get(ctx, req.CardNumber).Result()
	if err == redis.Nil {
		return nil, err
	} else if err != nil {
		log.Printf("Ошибка получения из Redis: %v", err)
		return nil, err
	} else {
		// Если данные найдены в кэше, возвращаем их в десериализованном виде
		var response rds.RedisGetCardResponse
		err := json.Unmarshal([]byte(cachedCardNumber), &response)
		if err != nil {
			log.Printf("Ошибка десериализации данных из Redis: %v", err)
			return nil, err
		}
		return &response, nil
	}
}

func (s *server) MakeRedisReplicate() {
	// Получаем максимальный размер памяти из переменной окружения
	//maxMemoryStr := os.Getenv("MAX_MEMORY")
	maxMemoryStr := "8589934592" // 8ГБ
	maxMemory, err := strconv.ParseInt(maxMemoryStr, 10, 64)
	if err != nil {
		log.Fatalf("Неверное значение MAX_MEMORY: %v", err)
	}

	ctx := context.Background()
	var usedMemory int64

	// Выполняем запрос к PostgreSQL
	rows, err := usfl.Db_redis_cache_server_conn.Query("SELECT user_id, card_type, card_number, card_expiry_date, availability, username, balance FROM cards")
	if err != nil {
		log.Fatalf("Ошибка при выполнении запроса: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cardData s.cardClient
		if err := rows.Scan(&cardData.UserId, &cardData.CardType, &cardData.CardNumber, &cardData.CardExpiryDate, &cardData.Availability, &cardData.Username, &cardData.Balance); err != nil {
			log.Fatalf("Ошибка при сканировании строки: %v", err)
		}

		// Преобразуем данные в JSON
		cardJson, err := json.Marshal(cardData)
		if err != nil {
			log.Fatalf("Ошибка при маршалинге JSON: %v", err)
		}

		// Проверяем, не превышает ли память
		if usedMemory+int64(len(cardJson)) > maxMemory {
			fmt.Println("Достигнут лимит памяти, прекращение репликации.")
			break
		}

		// Записываем данные в Redis
		if err := rdb.Set(ctx, cardData.CardNumber, cardJson, 0).Err(); err != nil {
			log.Fatalf("Ошибка при записи в Redis: %v", err)
		}

		usedMemory += int64(len(cardJson))
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Ошибка при обработке строк: %v", err)
	}

	fmt.Println("Данные успешно реплицированы!")
}

func main() {
	startGRPCserver()
}
