package consumer

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func TestNewReportQueueConsumer(t *testing.T) {
	log := zap.NewNop()
	consumer := NewReportQueueConsumer(nil, nil, log)

	if consumer == nil {
		t.Fatal("expected consumer")
	}
	if consumer.log != log {
		t.Error("expected logger to be set")
	}
}

func TestRun_ReturnsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deliveries := make(chan amqp.Delivery)
	consumer := NewReportQueueConsumer(nil, nil, zap.NewNop())

	consumer.Run(ctx, deliveries)
}

func TestRun_ReturnsWhenDeliveriesChannelCloses(t *testing.T) {
	ctx := context.Background()
	deliveries := make(chan amqp.Delivery)
	close(deliveries)
	consumer := NewReportQueueConsumer(nil, nil, zap.NewNop())

	consumer.Run(ctx, deliveries)
}
