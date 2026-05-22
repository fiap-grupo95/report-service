package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fiap/secure-systems/report-service/internal/domain"
	"github.com/fiap/secure-systems/report-service/internal/logging"
	"github.com/google/uuid"
)

type CreateReportInput struct {
	ProcessID   string
	Analysis    domain.Analysis
	RawResponse string
}

type CreateReportOutput struct {
	ReportID  string
	CreatedAt time.Time
}

type reportEvent struct {
	ProcessID string `json:"process_id"`
	ReportID  string `json:"report_id,omitempty"`
	Event     string `json:"event"`
	ErrorMsg  string `json:"error,omitempty"`
}

type CreateReportUseCase struct {
	repo        ReportRepository
	publisher   EventPublisher
	reportTopic string
}

func NewCreateReportUseCase(
	repo ReportRepository,
	publisher EventPublisher,
	reportTopic string,
) *CreateReportUseCase {
	return &CreateReportUseCase{repo: repo, publisher: publisher, reportTopic: reportTopic}
}

func (uc *CreateReportUseCase) Execute(ctx context.Context, in CreateReportInput) (*CreateReportOutput, error) {
	defer logging.StartSegment(ctx, "CreateReport.MongoSave")()

	report := &domain.Report{
		ID:          uuid.New().String(),
		ProcessID:   in.ProcessID,
		Analysis:    in.Analysis,
		RawResponse: in.RawResponse,
		CreatedAt:   time.Now().UTC(),
	}

	if err := uc.repo.Save(ctx, report); err != nil {
		uc.publishEvent(ctx, in.ProcessID, "", "report_failed", err.Error())
		return nil, fmt.Errorf("save report: %w", err)
	}

	uc.publishEvent(ctx, in.ProcessID, report.ID, "report_created", "")
	logging.LoggerWithContext(ctx).Info().
		Str("report_id", report.ID).Str("process_id", in.ProcessID).Msg("report created")

	return &CreateReportOutput{ReportID: report.ID, CreatedAt: report.CreatedAt}, nil
}

func (uc *CreateReportUseCase) publishEvent(ctx context.Context, processID, reportID, event, errMsg string) {
	payload, _ := json.Marshal(reportEvent{
		ProcessID: processID,
		ReportID:  reportID,
		Event:     event,
		ErrorMsg:  errMsg,
	})
	if err := uc.publisher.PublishToExchange(ctx, uc.reportTopic, payload); err != nil {
		logging.LoggerWithContext(ctx).Error().
			Str("event", event).Str("process_id", processID).Err(err).
			Msg("failed to publish report event")
	}
}
