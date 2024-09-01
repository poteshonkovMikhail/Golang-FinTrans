package database_methods

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // Импортируем драйвер PostgreSQL
)

type SomethingNeedInConnection interface {
	Connector()
}

type Postgres struct {
	Host     string
	Port     string
	User     string
	Password string
	Dbname   string
}

func (*Postgres) Connector(db *Postgres) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		db.Host, db.Port, db.User, db.Password, db.Dbname)

	database, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("could not open db connection: %v", err)
	}

	// Проверяем соединение
	if err := database.Ping(); err != nil {
		database.Close()
		return nil, fmt.Errorf("could not ping db: %v", err)
	}

	return database, nil
}
