package entity

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID          uuid.UUID
	UserOrderID string
	CreatedAt   time.Time `json:"created_at"`
}
