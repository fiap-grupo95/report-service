package queue

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("amqp dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("amqp channel: %w", err)
	}
	return &RabbitMQ{conn: conn, ch: ch}, nil
}

func (r *RabbitMQ) Close() {
	r.ch.Close()
	r.conn.Close()
}

func (r *RabbitMQ) DeclareQueue(name string) error {
	_, err := r.ch.QueueDeclare(name, true, false, false, false, nil)
	return err
}

func (r *RabbitMQ) DeclareExchange(name string) error {
	return r.ch.ExchangeDeclare(name, "fanout", true, false, false, false, nil)
}

func (r *RabbitMQ) PublishToExchange(ctx context.Context, exchange string, payload []byte) error {
	return r.ch.PublishWithContext(ctx, exchange, "", false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         payload,
	})
}

func (r *RabbitMQ) Consume(queue string) (<-chan amqp.Delivery, error) {
	return r.ch.Consume(queue, "", false, false, false, false, nil)
}
