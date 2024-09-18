package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	usfl "fin-trans/database_methods_package"
	models "fin-trans/models_package"
	cardpb "fin-trans/proto/proto_generated/cards_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	cardpb.UnimplementedCardServiceServer
	mu    sync.Mutex
	cards map[string]models.Card
}

func generateCardNumber() string {
	return fmt.Sprintf("%016d", rand.Intn(10000000000000000))
}

func generateExpiryDate() string {
	now := time.Now()
	return fmt.Sprintf("%02d-%02d-%02d", now.Year()+5, now.Month(), now.Day())
}

func (s *server) CreateCard(ctx context.Context, req *cardpb.CreateCardRequest) (*cardpb.CreateCardResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cardNumber := generateCardNumber()
	cardExpiryDate := generateExpiryDate()

	newCard := models.Card{
		CardType:       req.CardType,
		CardNumber:     cardNumber,
		CardExpiryDate: cardExpiryDate,
		Availability:   true,
		Username:       req.Username,
	}

	// Сохраняем новую карту в sql базе данных
	_, err := usfl.DB.Exec("INSERT INTO cards (card_type, card_number, card_expiry_date, availability, username) VALUES ($1, $2, $3, $4, $5)",
		newCard.CardType, newCard.CardNumber, newCard.CardExpiryDate, newCard.Availability, newCard.Username)

	if err != nil {
		return nil, err
	}

	return &cardpb.CreateCardResponse{
		CardNumber: cardNumber,
		Message:    "Card created successfully",
	}, nil
}

func (s *server) getCardInfo(cardNumber string) (models.Card, error) {
	var cardInfo models.Card

	row := usfl.DB.QueryRow("SELECT user_id, card_type, card_number, card_expiry_date, availability, username, balance FROM cards WHERE card_number = $1", cardNumber)
	err := row.Scan(&cardInfo.UserID, &cardInfo.CardType, &cardInfo.CardNumber, &cardInfo.CardExpiryDate, &cardInfo.Availability, &cardInfo.Username, &cardInfo.Balance)

	if err != nil {
		if err == sql.ErrNoRows {
			return cardInfo, nil // Если запись не найдена, возвращаем пустую карту
		}
		return cardInfo, err // Возвращаем ошибку
	}

	return cardInfo, nil
}

func (s *server) GetCard(ctx context.Context, req *cardpb.GetCardRequest) (*cardpb.GetCardResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	card, err := s.getCardInfo(req.CardNumber)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении карты %v", err)
	}

	return &cardpb.GetCardResponse{
		UserId:         card.UserID,
		CardType:       card.CardType,
		CardNumber:     card.CardNumber,
		CardExpiryDate: card.CardExpiryDate,
		Availability:   card.Availability,
		Username:       card.Username,
		Balance:        card.Balance,
	}, nil
}

func (s *server) ListCards(ctx context.Context, req *cardpb.ListCardsRequest) (*cardpb.ListCardsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := usfl.DB.Query("SELECT user_id, card_type, card_number, card_expiry_date, availability, username FROM cards WHERE user_id = $1", req.UserId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cardList []*cardpb.GetCardResponse
	for rows.Next() {
		var card models.Card
		if err := rows.Scan(&card.UserID, &card.CardType, &card.CardNumber, &card.CardExpiryDate, &card.Availability, &card.Username); err != nil {
			return nil, err
		}
		cardList = append(cardList, &cardpb.GetCardResponse{
			UserId:         card.UserID,
			CardType:       card.CardType,
			CardNumber:     card.CardNumber,
			CardExpiryDate: card.CardExpiryDate,
			Availability:   card.Availability,
			Username:       card.Username,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &cardpb.ListCardsResponse{Cards: cardList}, nil
}

func startGRPCserver() {
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
	if err := connPostgres.DbConnector(); err != nil {
		fmt.Println("Error connecting to the database:", err)
	} else {
		fmt.Println("Successfully connected to the database!")
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	cardpb.RegisterCardServiceServer(s, &server{cards: make(map[string]models.Card)})
	reflection.Register(s)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

func main() {
	startGRPCserver()
}
