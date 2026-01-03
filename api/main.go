package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"order_processing/constants"
	"order_processing/entity"

	"order_processing/rabbitmq"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/gofiber/fiber/v2"
)

func main() {
	conn := rabbitmq.RabbitMQSetup()
	defer conn.Close()
	ch := rabbitmq.GetChannel(conn)
	defer ch.Close()

	// make exchange user order
	err := ch.ExchangeDeclare(constants.ExchangeUserOrderDirect, "direct", true, false, false, false, nil)
	rabbitmq.FailOnError(err, "can't create exchange user order")

	// make exchange payment
	err = ch.ExchangeDeclare(constants.ExchangePaymentDirect, "direct", true, false, false, false, nil)
	rabbitmq.FailOnError(err, "can't create exchange payment")

	// // make exchange stock
	// err = ch.ExchangeDeclare(constants.ExchangeStockBroadcast, "fanout", true, false, false, false, nil)
	// rabbitmq.FailOnError(err, "can't create exchange stock")

	app := fiber.New()
	app.Post("/order", handleOrder(ch))

	app.Listen("localhost:8000")
}

func handleOrder(ch *amqp.Channel) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// get incoming order request
		var userOrderRequest entity.UserOrderRequest
		err := ctx.BodyParser(&userOrderRequest)
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid json",
			})
		}

		// publish to rabbitmq
		userOrderID, err := uuid.NewV7()
		if err != nil {
			panic(err)
		}

		// create user order id and passing it to create user order and payment
		userOrderRequest.ID = userOrderID
		body, err := json.Marshal(userOrderRequest)
		rabbitmq.FailOnError(err, "unable to marshalling data")

		// create order
		go CreateOrder(ch, body)

		// add payment
		go AddPayment(ch, userOrderID.String())

		// // update stock
		// go UpdateStock(ch, reqCtx, body)

		// return response back to client
		return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"message": "user order created",
		})
	}
}

func CreateOrder(ch *amqp.Channel, body []byte) {
	reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := ch.PublishWithContext(reqCtx,
		constants.ExchangeUserOrderDirect,
		constants.RoutingKeyUserOrder,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         []byte(body),
			DeliveryMode: amqp.Persistent,
		})
	rabbitmq.FailOnError(err, "Failed to publish a message")

	log.Printf("Create Order: [x] Sent %s", body)
}

func AddPayment(ch *amqp.Channel, userOrderID string) {
	reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	body, err := json.Marshal(userOrderID)
	if err != nil {
		panic(err)
	}
	err = ch.PublishWithContext(reqCtx,
		constants.ExchangePaymentDirect,
		constants.RoutingKeyPayment,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         []byte(body),
			DeliveryMode: amqp.Persistent,
		})
	rabbitmq.FailOnError(err, "Failed to publish a message")

	log.Printf("Add Payment: [x] Sent %s", body)
}

func UpdateStock(ch *amqp.Channel, reqCtx context.Context, body []byte) {
	err := ch.PublishWithContext(reqCtx,
		constants.ExchangeStockBroadcast,
		"",
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         []byte(body),
			DeliveryMode: amqp.Persistent,
		})
	rabbitmq.FailOnError(err, "Failed to publish a message")

	log.Printf("Update Stock: [x] Sent %s", body)
}
