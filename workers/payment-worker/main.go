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

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	conn := rabbitmq.RabbitMQSetup()
	defer conn.Close()
	ch := rabbitmq.GetChannel(conn)
	defer ch.Close()

	// make exchange user order
	err := ch.ExchangeDeclare(constants.ExchangePaymentDirect, "direct", true, false, false, false, nil)
	rabbitmq.FailOnError(err, "can't create exchange user order")

	// make userOrder queue
	paymentQueue, err := ch.QueueDeclare(constants.PaymentQueue, true, false, false, false, nil)
	rabbitmq.FailOnError(err, "can't create user queue")

	// bind user queue to user order exchange
	err = ch.QueueBind(paymentQueue.Name, constants.RoutingKeyPayment, constants.ExchangePaymentDirect, false, nil)
	rabbitmq.FailOnError(err, "can't bind user queue to user order exchange")
	listenUserOrder(ch, paymentQueue)
}

func listenUserOrder(ch *amqp.Channel, paymentQueue amqp.Queue) {
	msgs, err := ch.Consume(
		paymentQueue.Name, // queue
		"",                // consumer
		false,             // auto ack
		false,             // exclusive
		false,             // no local
		false,             // no wait
		nil,               // args
	)
	rabbitmq.FailOnError(err, "Failed to register a consumer")
	var forever chan struct{}

	go func() {
		for d := range msgs {
			log.Printf(" [x] %s", d.Body)

			var userOrderID string
			err := json.Unmarshal(d.Body, &userOrderID)
			rabbitmq.FailOnError(err, "unable to unmarshal payment")

			// payment logic for waiting 5 seconds
			time.Sleep(5 * time.Second)

			// 6/10 for success
			// 3/10 for success
			// 1/10 for success
			score := getRandomScore()
			if score > 6 {
				// create payment recored
				var payment entity.Payment
				payment.ID, err = uuid.NewV7()
				if err != nil {
					panic(err)
				}
				payment.UserOrderID = userOrderID
				payment.CreatedAt = time.Now()

				db := client.PostgresClient(constants.Username, constants.Password, constants.Host, constants.Port, constants.DBName)
				orderRepository := repository.NewOrderRepository(db)
				err = orderRepository.InsertPayment(context.Background(), &payment)
				if err != nil {
					panic(err)
				}
				log.Println("user order id:", payment.UserOrderID)

				// update user order table
				log.Println("payment successfully inserted")
				err = orderRepository.UpdateStatusUserOrder(context.Background(), payment.UserOrderID, entity.StatusPurchased)
				if err != nil {
					panic(err)
				}
				log.Println("updated user order to purchased")
				d.Ack(false)
			} else {
				d.Nack(false, true)
			}

		}
	}()

	log.Printf(" [*] Waiting for logs. To exit press CTRL+C")
	<-forever
}

func getRandomScore() int {
	return 8
}
