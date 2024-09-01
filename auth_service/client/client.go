package main

import (
	"context"
	"fmt"
	"log"

	//"time"

	authpb "fin-trans/auth_service/proto" // Замените на свой путь к сгенерированным файлам

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := authpb.NewAuthServiceClient(conn)

	// Регистрация пользователя
	regResp, err := client.Register(context.Background(), &authpb.RegisterRequest{
		Username: "testuser",
		Password: "testpassword",
	})
	if err != nil {
		log.Fatalf("error during registration: %v", err)
	}
	fmt.Println("Register response:", regResp)

	// Аутентификация пользователя
	logResp, err := client.Login(context.Background(), &authpb.LoginRequest{
		Username: "testuser",
		Password: "testpassword",
	})
	if err != nil {
		log.Fatalf("error during login: %v", err)
	}
	fmt.Println("Login response:", logResp)
}
