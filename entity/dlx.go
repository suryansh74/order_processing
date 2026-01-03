// entity/dlx.go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type DLX struct {
	ID              uuid.UUID `json:"id"`
	PaymentID       string    `json:"payment_id"` // Foreign key (no constraint for now)
	NumberOfRetries int       `json:"number_of_retries"`
	IsReplayed      bool      `json:"is_replayed"`
	ServiceName     string    `json:"service_name"`
	Error           string    `json:"error"`
	CreatedAt       time.Time `json:"created_at"`
}
