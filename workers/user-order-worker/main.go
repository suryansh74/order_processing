package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"order_processing/client"
	"order_processing/constants"
	"order_processing/entity"
	"order_processing/repository"

	"order_processing/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	conn := rabbitmq.RabbitMQSetup()
	defer conn.Close()
	ch := rabbitmq.GetChannel(conn)
	defer ch.Close()

	// make exchange user order
	err := ch.ExchangeDeclare(constants.ExchangeUserOrderDirect, "direct", true, false, false, false, nil)
	rabbitmq.FailOnError(err, "can't create exchange user order")

	// make userOrder queue
	userQueue, err := ch.QueueDeclare(constants.UserOrderQueue, true, false, false, false, nil)
	rabbitmq.FailOnError(err, "can't create user queue")

	// bind user queue to user order exchange
	err = ch.QueueBind(userQueue.Name, constants.RoutingKeyUserOrder, constants.ExchangeUserOrderDirect, false, nil)
	rabbitmq.FailOnError(err, "can't bind user queue to user order exchange")
	listenUserOrder(ch, userQueue)
}

func listenUserOrder(ch *amqp.Channel, userQueue amqp.Queue) {
	msgs, err := ch.Consume(
		userQueue.Name, // queue
		"",             // consumer
		true,           // auto ack
		false,          // exclusive
		false,          // no local
		false,          // no wait
		nil,            // args
	)
	rabbitmq.FailOnError(err, "Failed to register a consumer")
	var forever chan struct{}

	go func() {
		for d := range msgs {
			log.Printf(" [x] %s", d.Body)
			var userOrderRequest entity.UserOrderRequest
			err := json.Unmarshal(d.Body, &userOrderRequest)
			rabbitmq.FailOnError(err, "unable to unmarshal user order request")

			var userOrder entity.UserOrder
			log.Println("user order id in create service", userOrderRequest.ID)
			userOrder.ID = userOrderRequest.ID
			userOrder.UserID = userOrderRequest.UserID
			userOrder.ProductID = userOrderRequest.ProductID
			userOrder.CreatedAt = time.Now()
			userOrder.Location = userOrderRequest.Location
			userOrder.Status = entity.StatusPending
			userOrder.Quantity = userOrderRequest.Quantity

			db := client.PostgresClient(constants.Username, constants.Password, constants.Host, constants.Port, constants.DBName)
			orderRepository := repository.NewOrderRepository(db)
			err = orderRepository.InsertUserOrder(context.Background(), &userOrder)
			if err != nil {
				panic(err)
			}
			log.Println("user order created successful")
		}
	}()

	log.Printf(" [*] Waiting for logs. To exit press CTRL+C")
	<-forever
}
