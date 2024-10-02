package models

import "time"

// Resource represents a computing resource offered by a Supplier.
type Resource struct {
    ID          string    `json:"id"`
    SupplierID  string    `json:"supplierId"`
    CPUCores    int       `json:"cpuCores"`
    Memory      int       `json:"memory"` // in GB
    Storage     int       `json:"storage"` // in GB
    GPU         string    `json:"gpu"`    // e.g., "NVIDIA GeForce RTX 3080"
    Bandwidth   int       `json:"bandwidth"` // in Mbps
    CostPerHour float64   `json:"costPerHour"`
    CreatedAt   time.Time `json:"createdAt"`
}

// Bid represents a bid made by a User for a Resource.
type Bid struct {
    ID         string    `json:"id"`
    UserID     string    `json:"userId"`
    ResourceID string    `json:"resourceId"`
    Amount     float64   `json:"amount"` // Bid amount per hour
    Duration   int       `json:"duration"` // in hours
    Status     string    `json:"status"` // e.g., "pending", "accepted", "rejected"
    CreatedAt  time.Time `json:"createdAt"`
}

// User represents a user who can bid for resources.
type User struct {
    ID        string    `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    Password  string    `json:"password"`
    CreatedAt time.Time `json:"createdAt"`
}

// Supplier represents a supplier who can offer resources.
type Supplier struct {
    ID        string    `json:"id"`
    CompanyName string    `json:"companyName"`
    Email     string    `json:"email"`
    Password  string    `json:"password"`
    CreatedAt time.Time `json:"createdAt"`
}