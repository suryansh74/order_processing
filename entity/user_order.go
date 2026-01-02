package entity

import (
	"time"

	"github.com/google/uuid"
)

type UserOrder struct {
	ID        uuid.UUID
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Location  string    `json:"location"`
	CreatedAt time.Time `json:"created_at"`
	Status    Status    `json:"status"`
}

type UserOrderRequest struct {
	UserID    string `json:"user_id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Location  string `json:"location"`
}
