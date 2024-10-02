package main

import (
		"context"
        "fmt"
        "log"
        "net/http"
        "os"

		"github.com/gunrgnhsr/Cycloud/pkg/db"
        "github.com/gunrgnhsr/Cycloud/pkg/handlers"
        "github.com/joho/godotenv"
)

func main() {
        // Load environment variables from .env
        err := godotenv.Load()
        if err != nil {
                log.Fatal("Error loading .env file")
        }

        // Get database configuration from environment variables
        dbConfig := pkg.DBConfig{
                Host:     os.Getenv("DB_HOST"),
                Port:     os.Getenv("DB_PORT"),
                User:     os.Getenv("DB_USER"),
                Password: os.Getenv("DB_PASS"),
                DBName:   os.Getenv("DB_NAME"),
        }

        db, err := pkg.NewDB(dbConfig)
        if err != nil {
                panic(err)
        }
        defer db.Close()

        // Set up routes and middleware
        http.HandleFunc("/resources", func(w http.ResponseWriter, r *http.Request) {
                // Add the database connection to the request context
                r = r.WithContext(context.WithValue(r.Context(), "db", db))

                // Call the appropriate handler based on the request method
                if r.Method == http.MethodPost {
                        handlers.CreateResource(w, r)
                } else if r.Method == http.MethodGet {
                        handlers.GetResource(w, r)
                } // ... handle other methods
        })

        // Start the server
        fmt.Println("Server listening on port 8080")
        log.Fatal(http.ListenAndServe(":8080", nil))
}