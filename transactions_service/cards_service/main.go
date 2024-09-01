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
	cardpb "fin-trans/transactions_service/proto/proto_generated/cards_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var Db *sql.DB

type server struct {
	cardpb.UnimplementedCardServiceServer
	mu    sync.Mutex
	cards map[string]Card
}

type Card struct {
	CardID       string
	UserID       string
	CardType     string
	CardNumber   string
	ExpiryDate   string
	Availability bool
	Username     string
	Balance      float64
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

	cardID := fmt.Sprintf("%d", rand.Int())
	cardNumber := generateCardNumber()
	expiryDate := generateExpiryDate()
	balance := 100.0

	newCard := Card{
		CardID:       cardID,
		UserID:       req.UserId,
		CardType:     req.CardType,
		CardNumber:   cardNumber,
		ExpiryDate:   expiryDate,
		Availability: true,
		Username:     req.Username,
		Balance:      balance,
	}

	s.cards[cardNumber] = newCard

	_, err := Db.Exec("INSERT INTO cards (card_type, card_number, card_expiry_date, availability, username) VALUES ($1, $2, $3, $4, $5)", req.CardType, cardNumber, expiryDate, true, req.Username)
	if err != nil {
		return nil, err
	}

	return &cardpb.CreateCardResponse{
		CardNumber: cardNumber,
		Message:    "Card created successfully",
	}, nil
}

func (s *server) getCardInfo(CardNumber string) (Card, error) {
	var cardInfo Card

	query := `SELECT 
    c.user_id,
    c.card_type,
    c.card_number,
    c.card_expiry_date,
    c.availability,
    c.username,
    ub.balance
FROM 
    cards c
JOIN 
    user_balances ub ON c.card_number = ub.card_number
WHERE 
    c.card_number = $1;`
	err := Db.QueryRow(query, CardNumber).Scan(&cardInfo.UserID, &cardInfo.CardType, &cardInfo.CardNumber, &cardInfo.ExpiryDate, &cardInfo.Availability, &cardInfo.Username, &cardInfo.Balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return cardInfo, err
		}
		return cardInfo, err
	}
	return cardInfo, nil
}

func (s *server) GetCard(ctx context.Context, req *cardpb.GetCardRequest) (*cardpb.GetCardResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	card, err := s.getCardInfo(req.CardNumber)
	if err != nil {
		return nil, fmt.Errorf("Ошибка при получении карты %v", err)
	}

	return &cardpb.GetCardResponse{
		CardId:       card.CardID,
		UserId:       card.UserID,
		CardType:     card.CardType,
		CardNumber:   card.CardNumber,
		ExpiryDate:   card.ExpiryDate,
		Availability: card.Availability,
		Username:     card.Username,
		Balance:      card.Balance, // Устанавливаем полученный баланс
	}, nil
}

func (s *server) ListCards(ctx context.Context, req *cardpb.ListCardsRequest) (*cardpb.ListCardsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var cardList []*cardpb.GetCardResponse
	for _, card := range s.cards {
		if card.UserID == req.UserId {
			cardList = append(cardList, &cardpb.GetCardResponse{
				CardId:       card.CardID,
				UserId:       card.UserID,
				CardType:     card.CardType,
				CardNumber:   card.CardNumber,
				ExpiryDate:   card.ExpiryDate,
				Availability: card.Availability,
				Username:     card.Username,
				Balance:      card.Balance,
			})
		}
	}

	return &cardpb.ListCardsResponse{Cards: cardList}, nil
}

func (s *server) DeleteCard(ctx context.Context, req *cardpb.DeleteCardRequest) (*cardpb.DeleteCardResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.cards[req.CardId]
	if !exists {
		return &cardpb.DeleteCardResponse{
			Success: false,
			Message: "Card not found",
		}, nil
	}

	delete(s.cards, req.CardId)
	return &cardpb.DeleteCardResponse{
		Success: true,
		Message: "Card deleted successfully",
	}, nil
}

func (s *server) CheckRecipientCard(ctx context.Context, req *cardpb.CheckRecipientCardRequest) (*cardpb.CheckRecipientCardResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	recipientCard, err := s.getCardInfo(req.RecipientCardNumber)
	if err != nil {
		return nil, err
	}

	if !recipientCard.Availability {
		return &cardpb.CheckRecipientCardResponse{
			Availability:      false,
			RecipientUsername: "",
		}, nil
	}
	return &cardpb.CheckRecipientCardResponse{
		Availability:      recipientCard.Availability,
		RecipientUsername: recipientCard.Username,
	}, nil
}

func main() {

	// Настройка подключения к базе данных
	var postgresClientMethods *usfl.Postgres
	db, err := postgresClientMethods.Connector(&usfl.Postgres{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "workout+5",
		Dbname:   "fintrans_transactions_postgres",
	})
	Db = db
	if err != nil {
		log.Fatalf("Ошибка при подключении к базе данных: %v", err)
	}
	defer db.Close()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	cardpb.RegisterCardServiceServer(s, &server{cards: make(map[string]Card)})
	reflection.Register(s)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
