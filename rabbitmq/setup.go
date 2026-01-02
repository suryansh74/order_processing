package rabbitmq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func RabbitMQSetup() *amqp.Connection {
	conn, err := amqp.Dial("amqp://guest:guest@serverhost:5672/")
	FailOnError(err, "Failed to connect to RabbitMQ")
	return conn
}

func GetChannel(conn *amqp.Connection) *amqp.Channel {
	ch, err := conn.Channel()
	FailOnError(err, "Failed to open a channel")
	return ch
}

func FailOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
