// repository/order_repository.go
package repository

import (
	"context"

	"order_processing/entity"

	"github.com/jackc/pgx/v5"
)

type OrderRepository interface {
	InsertUserOrder(ctx context.Context, userOrder *entity.UserOrder) error
	InsertPayment(ctx context.Context, payment *entity.Payment) error
	UpdateStatusUserOrder(ctx context.Context, userOrderID string, status entity.Status) error
	InsertDLX(ctx context.Context, dlx *entity.DLX) error // NEW
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

func (or *orderRepository) InsertPayment(ctx context.Context, payment *entity.Payment) error {
	query := `
        INSERT INTO payments (id, user_order_id, created_at) 
        VALUES ($1, $2, $3)
    `

	_, err := or.db.Exec(ctx, query,
		payment.ID,
		payment.UserOrderID,
		payment.CreatedAt,
	)
	return err
}

func (or *orderRepository) UpdateStatusUserOrder(ctx context.Context, userOrderID string, status entity.Status) error {
	_, err := or.db.Exec(ctx, "update user_orders set status=$1 where id=$2", status, userOrderID)
	return err
}

// NEW: Insert DLX record
func (or *orderRepository) InsertDLX(ctx context.Context, dlx *entity.DLX) error {
	query := `
        INSERT INTO dlx (id, payment_id, number_of_retries, is_replayed, service_name, error, created_at) 
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

	_, err := or.db.Exec(ctx, query,
		dlx.ID,
		dlx.PaymentID,
		dlx.NumberOfRetries,
		dlx.IsReplayed,
		dlx.ServiceName,
		dlx.Error,
		dlx.CreatedAt,
	)
	return err
}
