package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	pkg "github.com/gunrgnhsr/Cycloud/pkg/db"
	"github.com/gunrgnhsr/Cycloud/pkg/handlers"
)

func addDBToContext(db *sql.DB, r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), pkg.GetDBContextKey(), db))
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func main() {

	db, err := pkg.NewDB()
	if err != nil {
		panic(err)
	}

	muxRouter := mux.NewRouter()

	// Apply logging middleware
	muxRouter.Use(loggingMiddleware)

	// Set up routes and middleware
	muxRouter.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handlers.Login(w, addDBToContext(db, r))
	})

	muxRouter.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handlers.Logout(w, addDBToContext(db, r))
	})

	muxRouter.HandleFunc("/get-user-resources", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetUserResource(w, addDBToContext(db, r))
	})

	muxRouter.HandleFunc("/add-user-resources", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateResource(w, addDBToContext(db, r))
	})

	muxRouter.HandleFunc("/delete-user-resource/{rid}", func(w http.ResponseWriter, r *http.Request) {
		handlers.DeleteResource(w, addDBToContext(db, r))
	})

	muxRouter.HandleFunc("/update-resource-availability/{rid}", func(w http.ResponseWriter, r *http.Request) {
		handlers.UpdateResourceAvailability(w, addDBToContext(db, r))
	})

	muxRouter.HandleFunc("/available-resources/{rid}/{direction}", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetResources(w, addDBToContext(db, r))
	})

	muxRouter.HandleFunc("/get-resource/{rid}", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetLoanRequestResourceSpec(w, addDBToContext(db, r))
	})

	muxRouter.HandleFunc("/place-loan-request", func(w http.ResponseWriter, r *http.Request) {
		handlers.PlaceBid(w, addDBToContext(db, r))
	})

	muxRouter.HandleFunc("/get-loan-requests", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetUserBids(w, addDBToContext(db, r))
	})

	muxRouter.HandleFunc("/delete-loan-request/{bidId}", func(w http.ResponseWriter, r *http.Request) {
		handlers.RemoveUserBid(w, addDBToContext(db, r))
	})

        muxRouter.HandleFunc("/get-info", func(w http.ResponseWriter, r *http.Request) {
                handlers.GetUserInfo(w, addDBToContext(db, r))
        })

        muxRouter.HandleFunc("/add-credits", func(w http.ResponseWriter, r *http.Request) {
                handlers.AddCredits(w, addDBToContext(db, r))
        })

	// Create a server instance
	server := &http.Server{
		Addr:    ":8080",
		Handler: muxRouter,
	}

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
