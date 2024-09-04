package main

import (
	"context"
	"log"
	"time"

	cardpb "fin-trans/transactions_service/proto/proto_generated/cards_service"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := cardpb.NewCardServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Создание новой карты
	r, err := c.CreateCard(ctx, &cardpb.CreateCardRequest{Username: "Potes", CardType: "Debit"})
	if err != nil {
		log.Fatalf("could not create card: %v", err)
	}
	log.Printf("Created Card Number: %s", r.GetCardNumber())

	// Получение информации о карте
	card, err := c.GetCard(ctx, &cardpb.GetCardRequest{CardNumber: r.GetCardNumber()})
	if err != nil {
		log.Fatalf("could not get card: %v", err)
	}
	log.Printf("Card Info: %v", card)
}
