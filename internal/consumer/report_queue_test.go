package consumer

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestNewReportQueueConsumer(t *testing.T) {
	consumer := NewReportQueueConsumer(nil, nil)

	if consumer == nil {
		t.Fatal("expected consumer")
	}
}

func TestRun_ReturnsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deliveries := make(chan amqp.Delivery)
	consumer := NewReportQueueConsumer(nil, nil)

	consumer.Run(ctx, deliveries)
}

func TestRun_ReturnsWhenDeliveriesChannelCloses(t *testing.T) {
	ctx := context.Background()
	deliveries := make(chan amqp.Delivery)
	close(deliveries)
	consumer := NewReportQueueConsumer(nil, nil)

	consumer.Run(ctx, deliveries)
}
