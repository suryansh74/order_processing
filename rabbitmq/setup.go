package rabbitmq

import (
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func RabbitMQSetup() *amqp.Connection {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	var conn *amqp.Connection
	var err error
	for attempt := 1; attempt <= 15; attempt++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			log.Printf("connected to RabbitMQ at %s", url)
			return conn
		}
		log.Printf("RabbitMQ dial attempt %d/15 failed: %v (retry in 2s)", attempt, err)
		time.Sleep(2 * time.Second)
	}
	log.Panicf("Failed to connect to RabbitMQ after retries: %s", err)
	return nil
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
