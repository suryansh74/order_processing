package constants

const (
	// exhange
	ExchangeUserOrderDirect = "exchange_user_order_direct"
	ExchangePaymentDirect   = "exhange_payment_direct"
	ExchangeDLQ             = "exchange_dlq"
	ExchangeStockBroadcast  = "exchange_stock_broadcast"

	// routing key
	RoutingKeyUserOrder = "routing_key_user_order"
	RoutingKeyPayment   = "routing_key_payment"
	RoutingKeyRetry     = "routing_key_retry"

	// queue
	UserOrderQueue    = "user_order_queue"
	PaymentQueue      = "payment_queue"
	RetryQueue        = "retry_queue"
	InventoryQueue    = "inventory_queue"
	NotificationQueue = "notification_queue"

	// database postgres connection
	Username = "root"
	Password = "secret"
	Host     = "serverhost"
	Port     = "5432"
	DBName   = "order_processing_db"
)
