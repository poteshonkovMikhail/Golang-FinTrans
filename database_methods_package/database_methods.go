package database_methods

import (
	"database/sql"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/lib/pq"
)

var (
	Db *sql.DB
)

type Connector interface {
	DbConnector(conn *ConnPostgres) error
}

type ConnPostgres struct {
	Host     string
	User     string
	Password string
	DbName   string
	Port     string
	SslMode  string
}

func (s *ConnPostgres) DbConnector() error {
	// Формируем строку подключения
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		s.Host, s.User, s.Password, s.DbName, s.Port, s.SslMode)
	var err error
	// Подключаемся к базе данных
	Db, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("ошибка при подключении к базе данных: %w", err)
	}

	// Проверяем подключение
	if err = Db.Ping(); err != nil {
		return fmt.Errorf("не удалось установить соединение с базой данных: %w", err)
	}

	return nil
}

// Настройка подключения к базе данных с использованием GORM
func SetupGormDatabase() *gorm.DB {
	dsn := "host=localhost user=postgres password=workout+5 dbname=fintrans_transactions_postgres port=5432 sslmode=disable"
	Db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка при подключении к базе данных: %v", err)
	}
	return Db
}
