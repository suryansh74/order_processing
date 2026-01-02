package repository

import (
	"context"

	"order_processing/entity"

	"github.com/jackc/pgx/v5"
)

type OrderRepository interface {
	InsertUserOrder(ctx context.Context, userOrder *entity.UserOrder) error
	// UpdateStatusUserOrder(ctx context.Context, userOrderID string, status entity.Status)
}

type orderRepository struct {
	db *pgx.Conn
}

func NewOrderRepository(db *pgx.Conn) OrderRepository {
	return &orderRepository{
		db: db,
	}
}

func (or *orderRepository) InsertUserOrder(ctx context.Context, userOrder *entity.UserOrder) error {
	query := `
        INSERT INTO user_orders (id, user_id, product_id, quantity, location, status, created_at) 
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

	_, err := or.db.Exec(ctx, query,
		userOrder.ID,
		userOrder.UserID,
		userOrder.ProductID,
		userOrder.Quantity,
		userOrder.Location,
		userOrder.Status,
		userOrder.CreatedAt,
	)
	return err
}
