package main

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"order_processing/constants"
	"order_processing/entity"
	"order_processing/rabbitmq"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ordersTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "orders_submitted_total",
		Help: "Total orders submitted via API",
	})
	publishTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rabbitmq_publish_total",
		Help: "Messages published to RabbitMQ",
	}, []string{"exchange", "status"})
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests",
	}, []string{"method", "path", "status"})
)

func main() {
	conn := rabbitmq.RabbitMQSetup()
	defer conn.Close()
	ch := rabbitmq.GetChannel(conn)
	defer ch.Close()

	err := ch.ExchangeDeclare(constants.ExchangeUserOrderDirect, "direct", true, false, false, false, nil)
	rabbitmq.FailOnError(err, "can't create exchange user order")

	err = ch.ExchangeDeclare(constants.ExchangePaymentDirect, "direct", true, false, false, false, nil)
	rabbitmq.FailOnError(err, "can't create exchange payment")

	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		err := c.Next()
		httpRequests.WithLabelValues(c.Method(), c.Path(), strconv.Itoa(c.Response().StatusCode())).Inc()
		return err
	})

	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Post("/order", handleOrder(ch))

	log.Println("API listening on :8000")
	app.Listen("0.0.0.0:8000")
}

func handleOrder(ch *amqp.Channel) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		var userOrderRequest entity.UserOrderRequest
		if err := ctx.BodyParser(&userOrderRequest); err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
		}

		userOrderID, err := uuid.NewV7()
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "id generation failed"})
		}

		userOrderRequest.ID = userOrderID
		body, err := json.Marshal(userOrderRequest)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "marshal failed"})
		}

		go CreateOrder(ch, body)
		go AddPayment(ch, userOrderID.String())

		ordersTotal.Inc()
		return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{"message": "user order created", "id": userOrderID.String()})
	}
}

func CreateOrder(ch *amqp.Channel, body []byte) {
	reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := ch.PublishWithContext(reqCtx,
		constants.ExchangeUserOrderDirect,
		constants.RoutingKeyUserOrder,
		false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})
	if err != nil {
		publishTotal.WithLabelValues(constants.ExchangeUserOrderDirect, "error").Inc()
		log.Printf("Create Order publish error: %v", err)
		return
	}
	publishTotal.WithLabelValues(constants.ExchangeUserOrderDirect, "ok").Inc()
	log.Printf("Create Order: [x] Sent %s", body)
}

func AddPayment(ch *amqp.Channel, userOrderID string) {
	reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	body, err := json.Marshal(userOrderID)
	if err != nil {
		log.Printf("Add Payment marshal error: %v", err)
		return
	}
	err = ch.PublishWithContext(reqCtx,
		constants.ExchangePaymentDirect,
		constants.RoutingKeyPayment,
		false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})
	if err != nil {
		publishTotal.WithLabelValues(constants.ExchangePaymentDirect, "error").Inc()
		log.Printf("Add Payment publish error: %v", err)
		return
	}
	publishTotal.WithLabelValues(constants.ExchangePaymentDirect, "ok").Inc()
	log.Printf("Add Payment: [x] Sent %s", body)
}
