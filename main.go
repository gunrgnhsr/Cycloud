package main

import (
	"context"
        "fmt"
        "log"
        "net/http"
        "os"
        "os/signal"
        "syscall"
        "time"

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

        // Set up routes and middleware        
        http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
                r = r.WithContext(context.WithValue(r.Context(), "db", db))
                handlers.Login(w, r)
        })

        http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
                r = r.WithContext(context.WithValue(r.Context(), "db", db))
                handlers.Logout(w, r)
        })
        
        http.HandleFunc("/resources", func(w http.ResponseWriter, r *http.Request) {
                // Add the database connection to the request context
                r = r.WithContext(context.WithValue(r.Context(), "db", db))

                // Call the appropriate handler based on the request method
                if r.Method == http.MethodPost {
                        handlers.CreateResource(w, r)
                } else if r.Method == http.MethodGet {
                        handlers.GetResource(w, r)
                } // ... handle other methods
        });

        // Create a server instance
        server := &http.Server{Addr: ":8080", Handler: nil}

        // Graceful shutdown
        go func() {
                stop := make(chan os.Signal, 1)
                signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
                <-stop
                log.Println("Shutting down server...")

                // Create a deadline for the server to shutdown gracefully
                ctx, cancel := context.WithTimeout(context.Background(), time.Second)
                defer cancel()

                if err := server.Shutdown(ctx); err != nil {
                        log.Fatalf("Server forced to shutdown: %v", err)
                }
        }()

        // Start the server
        fmt.Println("Server listening on port 8080")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                log.Fatalf("Server error: %v", err)
        }

        log.Println("Server exiting")
}