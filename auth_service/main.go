package main

import (
	"context"
	//"go/token"
	"sync"

	//"crypto/rand"
	//"encoding/base64"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authpb "fin-trans/auth_service/proto"
)

// Session хранит информацию о сессии пользователя
type Session struct {
	Token     string
	ExpiresAt time.Time
}

// Server реализует методы аутентификации
type server struct {
	authpb.UnimplementedAuthServiceServer
	users    map[string]string // пользователи и их хешированные пароли
	sessions map[string]*Session
	mu       sync.Mutex // для защиты map сессий
}

var secretKey = []byte("supersecretkey")

func (s *server) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[req.Username]; exists {
		return &authpb.RegisterResponse{Success: false, Message: "User already exists"}, nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not hash password")
	}

	s.users[req.Username] = string(hashedPassword)
	return &authpb.RegisterResponse{Success: true, Message: "User registered successfully"}, nil
}

func (s *server) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	hashedPassword, exists := s.users[req.Username]
	if !exists || bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)) != nil {
		return &authpb.LoginResponse{Success: false, Token: ""}, nil
	}

	token, err := generateToken(req.Username)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not generate token")
	}

	// Создание и сохранение сессии
	session := &Session{
		Token:     token,
		ExpiresAt: time.Now().Add(time.Minute * 1), // Время истечения токена 1 час
	}
	s.sessions[req.Username] = session

	return &authpb.LoginResponse{Success: true, Token: token}, nil
}

func (s *server) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, req.Username)
	return &authpb.LogoutResponse{Success: true, Message: "Logged out successfully"}, nil
}

func (s *server) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[req.Username]
	if !exists || session.Token != req.Token || session.ExpiresAt.Before(time.Now()) {
		return &authpb.ValidateTokenResponse{Valid: false}, nil
	}

	return &authpb.ValidateTokenResponse{Valid: true}, nil
}

func generateToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Minute * 1).Unix(), // Время истечения токена 1 час
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	authService := &server{
		users:    make(map[string]string),
		sessions: make(map[string]*Session),
	}
	authpb.RegisterAuthServiceServer(s, authService)

	fmt.Println("Listening on port :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
