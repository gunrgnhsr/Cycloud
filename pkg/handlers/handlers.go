package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "io"

    "github.com/google/uuid"
    "github.com/gunrgnhsr/Cycloud/pkg/models"
)

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