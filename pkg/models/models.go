package models

import (
	"sync"
	"time"
)

// Resource represents a computing resource offered by a Supplier.
type Resource struct {
	CPUCores      int     `json:"cpuCores"`
	Memory        int     `json:"memory"`    // in GB
	Storage       int     `json:"storage"`   // in GB
	GPU           string  `json:"gpu"`       // e.g., "NVIDIA GeForce RTX 3080"
	Bandwidth     int     `json:"bandwidth"` // in Mbps
	CostPerMinute float64 `json:"costPerHour"`
	Available     bool    `json:"available"`
	Computing     bool    `json:"computing"`
}

// ResourceWithID represents a computing resource with an ID.
type ResourceWithID struct {
	RID string `json:"rid"`
	Resource
	CreatedAt time.Time `json:"createdAt"`
}

// ResourceWithUID represents a computing resource with a UID.
type ResourceWithUID struct {
	UID string `json:"uid"`
	ResourceWithID
}

// Bid represents a bid made by a User for a Resource.
type Bid struct {
	RID      string  `json:"rid"`
	Amount   float64 `json:"amount"`   // Bid amount per hour
	Duration int     `json:"duration"` // in hours
}

// BidWithID represents a bid with an ID.
type BidWithID struct {
	BID string `json:"bid"`
	Bid
	Status    string    `json:"status"` // e.g., "pending", "accepted", "rejected"
	Computing bool      `json:"computing"`
	CreatedAt time.Time `json:"createdAt"`
}

type BidWithUID struct {
	UID string `json:"uid"`
	BidWithID
}

// Credintials represents a user credintials.
type Credintials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CredintialsWithID struct {
	UID string `json:"uid"`
	Credintials
	CreatedAt time.Time `json:"createdAt"`
}

// Token represents a JWT token.
type Tokens struct {
	UID   string `json:"uid"`
	Token string `json:"token"`
}

type UserWallet struct {
	UID    string  `json:"uid"`
	Amount float64 `json:"amount"`
}

type BidWithLock struct {
	MaxBid BidWithID
	Lock   sync.Mutex
}
