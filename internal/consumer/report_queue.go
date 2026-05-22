package consumer

import (
	"context"
	"encoding/json"

	"github.com/fiap/secure-systems/report-service/internal/domain"
	"github.com/fiap/secure-systems/report-service/internal/logging"
	"github.com/fiap/secure-systems/report-service/internal/usecase"
	"github.com/newrelic/go-agent/v3/newrelic"
	amqp "github.com/rabbitmq/amqp091-go"
)

type reportMessage struct {
	ProcessID   string          `json:"process_id"`
	Analysis    domain.Analysis `json:"analysis"`
	RawResponse string          `json:"raw_response"`
}

type ReportQueueConsumer struct {
	uc    *usecase.CreateReportUseCase
	nrApp *newrelic.Application
}

func NewReportQueueConsumer(
	uc *usecase.CreateReportUseCase,
	nrApp *newrelic.Application,
) *ReportQueueConsumer {
	return &ReportQueueConsumer{uc: uc, nrApp: nrApp}
}

func (c *ReportQueueConsumer) Run(ctx context.Context, deliveries <-chan amqp.Delivery) {
	logging.Logger().Info().Msg("report queue consumer started")
	for {
		select {
		case <-ctx.Done():
			logging.Logger().Info().Msg("report queue consumer stopped")
			return
		case d, ok := <-deliveries:
			if !ok {
				logging.Logger().Warn().Msg("report queue channel closed")
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
		logging.Logger().Error().
			Err(err).
			Int("body_size", len(d.Body)).
			Msg("invalid report queue message")
		txn.NoticeError(err)
		d.Nack(false, false)
		return
	}

	if msg.ProcessID == "" {
		logging.Logger().Error().
			Int("body_size", len(d.Body)).
			Msg("report message missing process_id")
		d.Nack(false, false)
		return
	}

	ctx := newrelic.NewContext(context.Background(), txn)
	txn.AddAttribute("process_id", msg.ProcessID)

	log := logging.LoggerWithContext(ctx).With().
		Str("process_id", msg.ProcessID).
		Logger()

	out, err := c.uc.Execute(ctx, usecase.CreateReportInput{
		ProcessID:   msg.ProcessID,
		Analysis:    msg.Analysis,
		RawResponse: msg.RawResponse,
	})
	if err != nil {
		log.Error().Err(err).Msg("create report failed")
		txn.NoticeError(err)
		d.Nack(false, false)
		return
	}

	log.Info().Str("report_id", out.ReportID).Msg("report created successfully")
	d.Ack(false)
}
