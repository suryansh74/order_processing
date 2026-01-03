// main.go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"strings"
	"time"

	"order_processing/client"
	"order_processing/constants"
	"order_processing/entity"
	"order_processing/rabbitmq"
	"order_processing/repository"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Track payment attempt scenarios for simulation
var paymentScenarios = map[string]string{} // paymentID -> scenario type

func main() {
	// Seed random for simulation
	rand.Seed(time.Now().UnixNano())

	conn := rabbitmq.RabbitMQSetup()
	defer conn.Close()
	ch := rabbitmq.GetChannel(conn)
	defer ch.Close()

	// PAYMENT SETUP
	// =======================================================================================

	// make exchange payment
	err := ch.ExchangeDeclare(constants.ExchangePaymentDirect, "direct", true, false, false, false, nil)
	rabbitmq.FailOnError(err, "can't create exchange user order")

	// FIXED: x-dead-letter arguments should be in QueueDeclare, not QueueBind
	paymentQueue, err := ch.QueueDeclare(
		constants.PaymentQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    constants.ExchangeDLX,
			"x-dead-letter-routing-key": constants.RoutingKeyRetry,
		},
	)
	rabbitmq.FailOnError(err, "can't create payment queue")

	// bind payment queue to payment exchange (no extra args here)
	err = ch.QueueBind(paymentQueue.Name, constants.RoutingKeyPayment, constants.ExchangePaymentDirect, false, nil)
	rabbitmq.FailOnError(err, "can't bind payment queue to payment exchange")

	// DLQ SETUP
	// =======================================================================================

	// make exchange dlx
	err = ch.ExchangeDeclare(constants.ExchangeDLX, "direct", true, false, false, false, nil)
	rabbitmq.FailOnError(err, "can't create exchange dlx")

	// FIXED: x-dead-letter and x-message-ttl should be in QueueDeclare
	retryQueue, err := ch.QueueDeclare(
		constants.RoutingKeyRetry,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    constants.ExchangePaymentDirect,
			"x-dead-letter-routing-key": constants.RoutingKeyPayment,
			"x-message-ttl":             constants.RetryDelaySeconds * 1000, // 5000ms
		},
	)
	rabbitmq.FailOnError(err, "can't create retry queue")

	// bind dlx queue to dlx exchange (no extra args here)
	err = ch.QueueBind(retryQueue.Name, constants.RoutingKeyRetry, constants.ExchangeDLX, false, nil)
	rabbitmq.FailOnError(err, "can't bind retry queue to dlx exchange")

	log.Println("✅ Topology setup complete")
	log.Println("\n🎬 Payment Worker Ready!")
	log.Println("📋 Possible Scenarios:")
	log.Println("   1️⃣  SUCCESS: Payment succeeds on first attempt")
	log.Println("   2️⃣  RETRY: Payment fails initially, succeeds after retry")
	log.Println("   3️⃣  DLX: Payment fails all 3 retries, stored in DLX table\n")

	// Start listening
	listenUserOrder(ch, paymentQueue)
}

func listenUserOrder(ch *amqp.Channel, paymentQueue amqp.Queue) {
	msgs, err := ch.Consume(
		paymentQueue.Name, // queue
		"",                // consumer
		false,             // auto ack - MUST be false
		false,             // exclusive
		false,             // no local
		false,             // no wait
		nil,               // args
	)
	rabbitmq.FailOnError(err, "Failed to register a consumer")
	var forever chan struct{}

	go func() {
		for d := range msgs {
			retryCount := getRetryCount(d.Headers)
			attemptNum := retryCount + 1
			log.Printf("\n" + strings.Repeat("=", 80))
			log.Printf("📨 Received message: %s", d.Body)
			log.Printf("🔄 Retry count: %d (Attempt #%d/%d)", retryCount, attemptNum, constants.MaxRetries+1)

			var userOrderID string
			err := json.Unmarshal(d.Body, &userOrderID)
			if err != nil {
				log.Printf("❌ Unable to unmarshal payment: %v", err)
				d.Ack(false) // Acknowledge to remove malformed message
				continue
			}

			// create payment record
			payment, err := createPayment(userOrderID)
			if err != nil {
				log.Printf("❌ Failed to create payment: %v", err)
				d.Nack(false, false) // Don't requeue, send to DLX if configured
				continue
			}

			// call paymentService with proper error handling
			err = paymentService(payment, retryCount)

			if err != nil {
				log.Printf("❌ Payment failed: %v", err)

				// Check if max retries exceeded
				if retryCount >= constants.MaxRetries {
					log.Printf("🚫 Max retries (%d) reached. Storing in DLX table.", constants.MaxRetries)

					// Store DLX record
					storeDLXRecord(payment.ID.String(), retryCount, err)

					// Clean up scenario tracking
					delete(paymentScenarios, payment.ID.String())

					// Acknowledge to remove from queue
					d.Ack(false)
				} else {
					log.Printf("🔁 Rejecting message. Will retry after %d seconds...", constants.RetryDelaySeconds)
					// Reject with requeue=false to trigger DLX
					d.Reject(false)
				}
			} else {
				log.Printf("✅ Payment succeeded! 🎉")

				// Clean up scenario tracking
				delete(paymentScenarios, payment.ID.String())

				d.Ack(false)
			}
			log.Printf(strings.Repeat("=", 80) + "\n")
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

// Simulates different payment scenarios
func paymentService(payment *entity.Payment, retryCount int) error {
	log.Printf("💳 Starting payment service for payment ID: %s", payment.ID)

	// payment logic takes 4 seconds
	time.Sleep(4 * time.Second)

	// Get or assign scenario for this payment
	scenario, exists := paymentScenarios[payment.ID.String()]
	if !exists {
		// Randomly assign scenario for new payments
		scenarios := []string{"success", "retry", "dlx"}
		scenario = scenarios[rand.Intn(len(scenarios))]
		paymentScenarios[payment.ID.String()] = scenario
		log.Printf("🎲 Assigned scenario: %s", strings.ToUpper(scenario))
	}

	// Execute based on scenario
	switch scenario {
	case "success":
		// Case 1: SUCCESS - Succeeds on first attempt
		log.Println("✅ [SCENARIO: SUCCESS] Payment processed successfully!")
		return nil

	case "retry":
		// Case 2: RETRY - Fails initially, succeeds on 2nd attempt
		if retryCount < 1 {
			log.Println("⚠️  [SCENARIO: RETRY] Payment gateway timeout - will retry")
			return errors.New("payment service error: gateway timeout")
		}
		log.Println("✅ [SCENARIO: RETRY] Payment succeeded after retry!")
		return nil

	case "dlx":
		// Case 3: DLX - Fails all attempts
		log.Printf("❌ [SCENARIO: DLX] Payment failed (attempt %d/%d)", retryCount+1, constants.MaxRetries+1)
		if retryCount >= constants.MaxRetries {
			return errors.New("payment service error: persistent gateway failure - max retries exceeded")
		}
		return errors.New("payment service error: gateway unavailable")

	default:
		return errors.New("unknown scenario")
	}
}

func createPayment(userOrderID string) (*entity.Payment, error) {
	var payment entity.Payment
	var err error
	payment.ID, err = uuid.NewV7()
	if err != nil {
		return nil, err
	}
	payment.UserOrderID = userOrderID
	payment.CreatedAt = time.Now()

	db := client.PostgresClient(constants.Username, constants.Password, constants.Host, constants.Port, constants.DBName)
	orderRepository := repository.NewOrderRepository(db)
	defer db.Close(context.Background())

	err = orderRepository.InsertPayment(context.Background(), &payment)
	if err != nil {
		return nil, err
	}
	log.Println("💾 Payment record inserted with ID:", payment.ID)

	// update user order table
	err = orderRepository.UpdateStatusUserOrder(context.Background(), payment.UserOrderID, entity.StatusPurchased)
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func storeDLXRecord(paymentID string, retryCount int, err error) {
	db := client.PostgresClient(constants.Username, constants.Password, constants.Host, constants.Port, constants.DBName)
	orderRepository := repository.NewOrderRepository(db)
	defer db.Close(context.Background())

	dlx := &entity.DLX{
		ID:              uuid.Must(uuid.NewV7()),
		PaymentID:       paymentID,
		NumberOfRetries: retryCount,
		IsReplayed:      false,
		ServiceName:     "payment",
		Error:           err.Error(),
		CreatedAt:       time.Now(),
	}

	if err := orderRepository.InsertDLX(context.Background(), dlx); err != nil {
		log.Printf("⚠️ Failed to insert DLX record: %v", err)
	} else {
		log.Printf("💾 DLX record stored successfully!")
		log.Printf("   📝 DLX ID: %s", dlx.ID)
		log.Printf("   💳 Payment ID: %s", dlx.PaymentID)
		log.Printf("   🔄 Total Retries: %d", dlx.NumberOfRetries)
		log.Printf("   ⚠️  Error: %s", dlx.Error)
	}
}

func getRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}

	xDeath, ok := headers["x-death"]
	if !ok {
		return 0
	}

	// x-death is an array of tables
	xDeathArray, ok := xDeath.([]any)
	if !ok || len(xDeathArray) == 0 {
		return 0
	}

	// Get the first entry (most recent death)
	firstDeath, ok := xDeathArray[0].(amqp.Table)
	if !ok {
		return 0
	}

	// Extract count field
	count, ok := firstDeath["count"].(int64)
	if !ok {
		return 0
	}

	return int(count)
}
