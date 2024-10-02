package pkg

import (
        "database/sql"
        "fmt"
		_ "github.com/lib/pq"
)

// DBConfig holds the database configuration parameters.
type DBConfig struct {
        Host     string
        Port     string
        User     string
        Password string
        DBName   string
}

// NewDB creates a new database connection.
func NewDB(cfg DBConfig) (*sql.DB, error) {
        connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
                cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)

        db, err := sql.Open("postgres", connStr)
        if err != nil {
                return nil, err
        }

        err = db.Ping()
        if err != nil {
                return nil, err
        }

        return db, nil
}