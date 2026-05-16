package consumer

import (
	"context"
	"encoding/json"

	"github.com/fiap/secure-systems/report-service/internal/domain"
	"github.com/fiap/secure-systems/report-service/internal/usecase"
	"github.com/newrelic/go-agent/v3/newrelic"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type reportMessage struct {
	ProcessID   string          `json:"process_id"`
	Analysis    domain.Analysis `json:"analysis"`
	RawResponse string          `json:"raw_response"`
}

type ReportQueueConsumer struct {
	uc    *usecase.CreateReportUseCase
	nrApp *newrelic.Application
	log   *zap.Logger
}

func NewReportQueueConsumer(
	uc *usecase.CreateReportUseCase,
	nrApp *newrelic.Application,
	log *zap.Logger,
) *ReportQueueConsumer {
	return &ReportQueueConsumer{uc: uc, nrApp: nrApp, log: log}
}

func (c *ReportQueueConsumer) Run(ctx context.Context, deliveries <-chan amqp.Delivery) {
	c.log.Info("report queue consumer started")
	for {
		select {
		case <-ctx.Done():
			c.log.Info("report queue consumer stopped")
			return
		case d, ok := <-deliveries:
			if !ok {
				c.log.Warn("report queue channel closed")
				return
			}
			c.handle(d)
		}
	}
}

func (c *ReportQueueConsumer) handle(d amqp.Delivery) {
	txn := c.nrApp.StartTransaction("consumer/report-queue")
	defer txn.End()

	var msg reportMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		c.log.Error("invalid report queue message", zap.Error(err))
		txn.NoticeError(err)
		d.Nack(false, false)
		return
	}

	if msg.ProcessID == "" {
		c.log.Error("report message missing process_id")
		d.Nack(false, false)
		return
	}

	ctx := newrelic.NewContext(context.Background(), txn)
	txn.AddAttribute("processId", msg.ProcessID)

	_, err := c.uc.Execute(ctx, usecase.CreateReportInput{
		ProcessID:   msg.ProcessID,
		Analysis:    msg.Analysis,
		RawResponse: msg.RawResponse,
	})
	if err != nil {
		c.log.Error("create report failed",
			zap.String("processId", msg.ProcessID),
			zap.Error(err),
		)
		txn.NoticeError(err)
		d.Nack(false, false)
		return
	}

	d.Ack(false)
}
