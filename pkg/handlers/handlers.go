package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gunrgnhsr/Cycloud/pkg/auth"
	pkg "github.com/gunrgnhsr/Cycloud/pkg/db"
	"github.com/gunrgnhsr/Cycloud/pkg/models"
)

// get db connection from context
func getDB(r *http.Request) *sql.DB {
	return r.Context().Value(pkg.GetDBContextKey()).(*sql.DB)
}

// handleCORS sets the CORS headers in the response
func handleCORS(w http.ResponseWriter, r *http.Request, headers string, methods string) bool {
	// Get the Origin header from the request
	origin := r.Header.Get("Origin")

	// Set CORS headers in the response
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", methods)
	w.Header().Set("Access-Control-Allow-Headers", headers+", OPTIONS")

	// Handle preflight requests
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK) // Return 200 OK for preflight requests
		return true
	}

	return false
}

// checkAuthorization checks if the request is authorized by validating the JWT token
func checkAuthorization(r *http.Request) (string, error) {
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
	db := getDB(r)

	// Check if the token exists in the tokens table
	uid, err := pkg.GetUserIDFromToken(db, tokenString)
	if err != nil {
		return "", err
	}

	// Check if the token is expired
	if claims.ExpiresAt < time.Now().Unix() {
		// Remove the expired token from the database
		pkg.RemoveExpiredToken(db, tokenString)
	}

	// Return the username in the response
	return uid, nil
}

func checkThatResourceBelongsToUser(r *http.Request, uid string, rid string) error {
	// Get the database connection from the request context
	db := getDB(r)

	// Check if the resource belongs to the user
	ownerUID, err := pkg.GetResourceOwner(db, rid)
	if err != nil {
		return err
	}

	if ownerUID != uid {
		return errors.New("resource does not belong to user")
	}

	return nil
}

func checkThatBidBelongsToUser(r *http.Request, uid string, bidId string) error {
	// Get the database connection from the request context
	db := getDB(r)

	// Check if the bid belongs to the user
	ownerUID, err := pkg.GetBidOwner(db, bidId)
	if err != nil {
		return err
	}

	if ownerUID != uid {
		return errors.New("bid does not belong to user")
	}

	return nil
}

func checkThatTheresABidForTheResourceByUser(r *http.Request, rid string, uid string) error {
	// Get the database connection from the request context
	db := getDB(r)

	// Check if the bid belongs to the user
	thereIsBidForResource, err := pkg.CheckOwnerHaveBidForResource(db, uid, rid)
	if err != nil {
		return err
	}

	if !thereIsBidForResource {
		return errors.New("there is no bid for the resource by the user")
	}

	return nil
}

// Login handles user login and generates a JWT
func Login(w http.ResponseWriter, r *http.Request) {
	if handleCORS(w, r, "Authorization, content-type", "POST") {
		return
	}

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
	db := getDB(r)

	// Hash the username and password
	hashedUsername := auth.HashString(credentials.Username)

	hashedPassword := auth.HashString(credentials.Password)

	// Query the database to check if the user exists and the password matches
	uid, err := pkg.GetUserOrRegisterIfNotExist(db, hashedUsername, hashedPassword)
	if err != nil {
		if err.Error() == "failed to authenticate user" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else if err.Error() == "invalid username or password" {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		} else if err.Error() == "failed to register user" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if err.Error() == "user registered" {
			// w.WriteHeader(http.StatusCreated)
		} else {
			return
		}
	}

	if uid == "" {
		http.Error(w, "Failed to authenticate or register user", http.StatusUnauthorized)
		return
	}

	// Generate a JWT token
	tokenString, err := auth.GenerateJWT(hashedUsername, "client")
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Insert the generated token into the database
	err = pkg.InsertToken(db, uid, tokenString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the token in the response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"token": tokenString})
}

// Logout handles user logout by invalidating the JWT token
func Logout(w http.ResponseWriter, r *http.Request) {
	if handleCORS(w, r, "Authorization", "DELETE") {
		return
	}

	var (
		err error
		uid string
	)
	uid, err = checkAuthorization(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// TODO: Add the token to a blacklist or perform other invalidation logic
	// Get the database connection from the request context
	db := getDB(r)

	// Remove the token from the database
	err = pkg.RemoveExpiredToken(db, uid)
	if err != nil {
		if err.Error() != "failed to remove expired token" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// For example, you could store invalidated tokens in a database or in-memory store

	// Return a success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Logged out successfully"})
}

// CreateResource handles the creation of a new resource.
func CreateResource(w http.ResponseWriter, r *http.Request) {
	if handleCORS(w, r, "Authorization, content-type", "POST") {
		return
	}

	// Check if the request is authorized
	var (
		uid string
		err error
	)
	uid, err = checkAuthorization(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Parse the request body to get the resource details
	var resource models.Resource
	err = json.NewDecoder(r.Body).Decode(&resource)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get the database connection from the request context
	db := getDB(r)

	// Insert the resource into the database and return the generated ID
	err = pkg.InsertNewResourse(db, resource, uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return a success response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Resource created successfully"})
}

func UpdateResourceAvailability(w http.ResponseWriter, r *http.Request) {
	if handleCORS(w, r, "Authorization, content-type", "PUT") {
		return
	}

	// Check if the request is authorized
	var (
		uid string
		err error
	)
	uid, err = checkAuthorization(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Parse the request body to get the resource details
	rid := mux.Vars(r)["rid"]
	if rid == "" {
		http.Error(w, "Missing resource ID", http.StatusBadRequest)
		return
	}

	err = checkThatResourceBelongsToUser(r, uid, rid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Get the database connection from the request context
	db := getDB(r)

	// Update the resource availability in the database
	available, err := pkg.UpdateResourceAvailability(db, rid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !available {
		err = pkg.RemoveBidsForResource(db, rid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Return a success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Resource updated successfully"})
}

// DeleteResource handles the deletion of a resource by ID.
func DeleteResource(w http.ResponseWriter, r *http.Request) {
	if handleCORS(w, r, "Authorization, content-type", "DELETE") {
		return
	}

	// Check if the request is authorized
	var (
		uid string
		err error
	)
	uid, err = checkAuthorization(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Parse the request body to get the resource details
	rid := mux.Vars(r)["rid"]
	if rid == "" {
		http.Error(w, "Missing resource ID", http.StatusBadRequest)
		return
	}

	err = checkThatResourceBelongsToUser(r, uid, rid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Get the database connection from the request context
	db := getDB(r)

	available, err := pkg.CheckResourceAvailability(db, rid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if available {
		http.Error(w, "Resource is still available, please make it unavailable before removing", http.StatusPreconditionFailed)
		return
	}

	// Delete the resource from the database
	err = pkg.DeleteResource(db, rid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return a success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Resource deleted successfully"})
}

// GetResource handles the retrieval of a resource by ID.
func GetUserResource(w http.ResponseWriter, r *http.Request) {
	if handleCORS(w, r, "Authorization", "GET") {
		return
	}

	// Check if the request is authorized
	var (
		uid string
		err error
	)
	uid, err = checkAuthorization(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Get the database connection from the request context
	db := getDB(r)

	// Fetch the resource from the database
	resource, err := pkg.GetUserResources(db, uid)
	if err != nil {
		if err.Error() == "Failed to fetch resource" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if err.Error() == "Resource not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	}

	// Return the resource data
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resource)
}

// GetResources handles the retrieval of all resources.
func GetResources(w http.ResponseWriter, r *http.Request) {
	if handleCORS(w, r, "Authorization", "GET") {
		return
	}

	// Check if the request is authorized
	var err error
	_, err = checkAuthorization(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Get the database connection from the request context
	db := getDB(r)

	// Parse the request body to get the resource details
	rid := mux.Vars(r)["rid"]
	if rid == "" {
		http.Error(w, "Missing resource ID", http.StatusBadRequest)
		return
	}

	direction := mux.Vars(r)["direction"]
	if rid == "" {
		http.Error(w, "Missing direction", http.StatusBadRequest)
		return
	}

	var resources []models.ResourceWithID
	var isPrev bool
	if direction == "prev" {
		isPrev = true
	} else if direction == "next" {
		isPrev = false
	} else {
		http.Error(w, "Invalid direction", http.StatusBadRequest)
		return
	}

	resources, err = pkg.GetNextOrPrevTwentyAvailableResourcesFromGivenRID(db, rid, isPrev)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the resources data
	json.NewEncoder(w).Encode(resources)
}

// PlaceBid handles the placement of a bid on a resource.
func PlaceBid(w http.ResponseWriter, r *http.Request) {
	if handleCORS(w, r, "Authorization, content-type", "POST") {
		return
	}

	// Check if the request is authorized
	uid, err := checkAuthorization(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Parse the request body to get the bid details
	var bid models.Bid
	err = json.NewDecoder(r.Body).Decode(&bid)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get the database connection from the request context
	db := getDB(r)

	// Insert the bid into the database
	bidId, err := pkg.InsertNewBid(db, uid, bid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return a success response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Bid placed successfully", "bid_id": bidId})
}

// GetUserBids handles the retrieval of all bids.
func GetUserBids(w http.ResponseWriter, r *http.Request) {
	if handleCORS(w, r, "Authorization", "GET") {
		return
	}

	// Check if the request is authorized
	uid, err := checkAuthorization(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Get the database connection from the request context
	db := getDB(r)

	// Fetch the bids from the database
	bids, err := pkg.GetUserBids(db, uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the bids data
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bids)
}

// RemoveUserBid handles the removal of a bid by ID.
func RemoveUserBid(w http.ResponseWriter, r *http.Request) {
	if handleCORS(w, r, "Authorization", "DELETE") {
		return
	}

	// Check if the request is authorized
	uid, err := checkAuthorization(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Parse the request body to get the bid details
	bidId := mux.Vars(r)["bidId"]
	if bidId == "" {
		http.Error(w, "Missing bid ID", http.StatusBadRequest)
		return
	}

	// Check if the bid belongs to the user
	err = checkThatBidBelongsToUser(r, uid, bidId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Get the database connection from the request context
	db := getDB(r)

	// Remove the bid from the database
	err = pkg.RemoveBid(db, bidId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return a success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Bid removed successfully"})
}

// GetLoanRequestResourceSpec handles the retrieval of a resource by ID.
func GetLoanRequestResourceSpec(w http.ResponseWriter, r *http.Request) {
	if handleCORS(w, r, "Authorization", "GET") {
		return
	}

	// Check if the request is authorized
	uid, err := checkAuthorization(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Parse the request body to get the resource details
	rid := mux.Vars(r)["rid"]
	if rid == "" {
		http.Error(w, "Missing resource ID", http.StatusBadRequest)
		return
	}

	err = checkThatTheresABidForTheResourceByUser(r, rid, uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Get the database connection from the request context
	db := getDB(r)

	// Fetch the resource from the database
	resource, err := pkg.GetResourceByID(db, rid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the resource data
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resource)
}
