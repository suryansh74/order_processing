package entity

type Status string

const (
	StatusPending   Status = "pending"
	StatusPurchased Status = "purchased"
	StatusCancelled Status = "cancelled"
)
