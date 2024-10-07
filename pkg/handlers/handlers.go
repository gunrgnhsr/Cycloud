package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gunrgnhsr/Cycloud/pkg/auth"
	"github.com/gunrgnhsr/Cycloud/pkg/models"
)

// handleCORS sets the CORS headers in the response
func handleCORS(w http.ResponseWriter, r *http.Request, headers string) bool {
        // Get the Origin header from the request
        origin := r.Header.Get("Origin")

        // Set CORS headers in the response
        w.Header().Set("Access-Control-Allow-Origin", origin) 
        w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")                // Allow POST requests
        w.Header().Set("Access-Control-Allow-Headers", headers)        // Allow Content-Type header

        // Handle preflight requests
        if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusOK) // Return 200 OK for preflight requests
                return true
        }

        return false
}

// CheckAuthorization checks if the request is authorized by validating the JWT token
func CheckAuthorization(w http.ResponseWriter, r *http.Request) (string, error) {
        // Get the token from the Authorization header
        tokenString := r.Header.Get("Authorization")
        if tokenString == "" {
                return "", errors.New("missing token")
        }

        // Validate the token
        claims, err := auth.ValidateJWT(tokenString)
        if err != nil {
                return "", errors.New("invalid token")
        }

        // Get the database connection from the request context
        db := r.Context().Value("db").(*sql.DB)

        // Check if the token exists in the tokens table
        var username string
        err = db.QueryRow("SELECT username FROM tokens WHERE token = $1", tokenString).Scan(&username)
        if err != nil {
                if err == sql.ErrNoRows {
                        return "", errors.New("token not found")
                }
                return "", errors.New("failed to check token")
        }

        if username == "" {
                return "", errors.New("token not found")
        }

        // Check if the token is expired
        if claims.ExpiresAt < time.Now().Unix() {
                // Remove the expired token from the database
                _, err = db.Exec("DELETE FROM tokens WHERE token = $1", tokenString)
                if err != nil {
                        return "", errors.New("failed to remove expired token")
                }
                return "", errors.New("token expired")
        }

        // Return the username in the response
        return username, nil


}

// Login handles user login and generates a JWT
func Login(w http.ResponseWriter, r *http.Request) {
        handleCORS(w, r, "Authorization")
                
        // Parse the request body to get username and password
        var credentials struct {
                Username string `json:"username"`
                Password string `json:"password"`
        }
        err := json.NewDecoder(r.Body).Decode(&credentials)
        if err != nil {
                http.Error(w, "Invalid request body", http.StatusBadRequest)
                return
        }

        // Get the database connection from the request context
        db := r.Context().Value("db").(*sql.DB)

        // Query the database to check if the user exists and the password matches
        var storedPassword string
        err = db.QueryRow("SELECT password FROM users WHERE username = $1", credentials.Username).Scan(&storedPassword)
        if err != nil {
                if err == sql.ErrNoRows {
                        // Register the new user
                        _, err = db.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", credentials.Username, credentials.Password)
                        if err != nil {
                                http.Error(w, "Failed to register user", http.StatusInternalServerError)
                                return
                        }else{
                                w.WriteHeader(http.StatusCreated)
                        }
                } else {
                        http.Error(w, "Failed to authenticate user", http.StatusInternalServerError)
                        return
                }
        } else {
                // Compare the stored password with the provided password
                if storedPassword != credentials.Password {
                        http.Error(w, "Invalid username or password", http.StatusUnauthorized)
                        return
                }
        }

        // Generate a JWT token
        tokenString, err := auth.GenerateJWT(credentials.Username, "client")
        if err != nil {
                http.Error(w, "Failed to generate token", http.StatusInternalServerError)
                return
        }

        // Insert the generated token into the database
        _, err = db.Exec("INSERT INTO tokens (username, token) VALUES ($1, $2)", credentials.Username, tokenString)
        if err != nil {
                http.Error(w, "Failed to store token", http.StatusInternalServerError)
                return
        }

        // Return the token in the response
        json.NewEncoder(w).Encode(map[string]interface{}{"token": tokenString})
}

// Logout handles user logout by invalidating the JWT token
func Logout(w http.ResponseWriter, r *http.Request) {
        if handleCORS(w, r, "Authorization"){
                return
        }

        username, err := CheckAuthorization(w, r)
        if err != nil {
                http.Error(w, err.Error(), http.StatusUnauthorized)
                return
        }
        
        // TODO: Add the token to a blacklist or perform other invalidation logic
        // Get the database connection from the request context
        db := r.Context().Value("db").(*sql.DB)

        // Remove the token from the database
        _, err = db.Exec("DELETE FROM tokens WHERE username = $1", username)
        if err != nil {
                http.Error(w, "Failed to remove token", http.StatusInternalServerError)
                return
        }
        // For example, you could store invalidated tokens in a database or in-memory store

        // Return a success response
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]interface{}{"message": "Logged out successfully"})
}

// CreateResource handles the creation of a new resource.
func CreateResource(w http.ResponseWriter, r *http.Request) {
        // Parse the request body to get the resource details
        var resource models.Resource
        err := json.NewDecoder(r.Body).Decode(&resource)
        if err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
        }

        // Get the database connection from the request context
        db := r.Context().Value("db").(*sql.DB)

        // Insert the resource into the database
        result, err := db.Exec("INSERT INTO resources (id, supplier_id, cpu_cores, memory, storage, gpu, bandwidth, cost_per_hour) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
        resource.ID, resource.SupplierID, resource.CPUCores, resource.Memory, resource.Storage, resource.GPU, resource.Bandwidth, resource.CostPerHour)
        if err != nil {
        http.Error(w, "Failed to create resource", http.StatusInternalServerError)
        return
        }

        // Check if the resource was inserted
        rowsAffected, err := result.RowsAffected()
        if err != nil {
        http.Error(w, "Failed to create resource", http.StatusInternalServerError)
        return
        }
        if rowsAffected == 0 {
        http.Error(w, "Failed to create resource", http.StatusInternalServerError)
        return
        }

        // Return a success response
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(map[string]interface{}{"message": "Resource created successfully"})
}

// GetResource handles the retrieval of a resource by ID.
func GetResource(w http.ResponseWriter, r *http.Request) {
    // Get resource ID from request parameters
    resourceID := r.URL.Query().Get("id")
    if resourceID == "" {
            http.Error(w, "Missing resource ID", http.StatusBadRequest)
            return
    }

    // Get the database connection from the request context
    db := r.Context().Value("db").(*sql.DB)

    // Fetch the resource from the database
    var resource models.Resource
    err := db.QueryRow("SELECT id, supplier_id, cpu_cores, memory, storage, gpu, bandwidth, cost_per_hour FROM resources WHERE id = $1", resourceID).Scan(&resource.ID, &resource.SupplierID, &resource.CPUCores, &resource.Memory, &resource.Storage, &resource.GPU, &resource.Bandwidth, &resource.CostPerHour)
    if err != nil {
            if err == sql.ErrNoRows {
                    http.Error(w, "Resource not found", http.StatusNotFound)
            } else {
                    http.Error(w, "Failed to fetch resource", http.StatusInternalServerError)
            }
            return
    }

    // Return the resource data
    json.NewEncoder(w).Encode(resource)
}

// PlaceBid handles the placement of a bid on a resource.
func PlaceBid(w http.ResponseWriter, r *http.Request) {
    // Parse the request body to get the bid details
    var bid models.Bid
    body, err := io.ReadAll(r.Body)
    if err != nil {
            http.Error(w, "Failed to read request body", http.StatusBadRequest)
            return
    }
    err = json.Unmarshal(body, 
&bid)
    if err != nil {
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
    }

    // Get the database connection from the request context
    db := r.Context().Value("db").(*sql.DB)

    // Generate a UUID for the bid
    bid.ID = uuid.New().String()

    // Set the initial status of the bid
    bid.Status = "pending"

    // Insert the bid into the database
    result, err := db.Exec("INSERT INTO bids (id, user_id, resource_id, amount, duration, status) VALUES ($1, $2, $3, $4, $5, $6)",
            bid.ID, bid.UserID, bid.ResourceID, bid.Amount, bid.Duration, bid.Status)
    if err != nil {
            http.Error(w, "Failed to place bid", http.StatusInternalServerError)
            return
    }

    // Check if the bid was inserted
    rowsAffected, err := result.RowsAffected()
    if err != nil {
            http.Error(w, "Failed to place bid", http.StatusInternalServerError)
            return
    }
    if rowsAffected == 0 {
            http.Error(w, "Failed to place bid", http.StatusInternalServerError)
            return
    }

    // Return a success response
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]interface{}{"message": "Bid placed successfully", "bid_id": bid.ID})
}