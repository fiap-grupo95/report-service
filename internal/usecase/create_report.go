package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fiap/secure-systems/report-service/internal/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
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
	log         *zap.Logger
}

func NewCreateReportUseCase(
	repo ReportRepository,
	publisher EventPublisher,
	reportTopic string,
	log *zap.Logger,
) *CreateReportUseCase {
	return &CreateReportUseCase{repo: repo, publisher: publisher, reportTopic: reportTopic, log: log}
}

func (uc *CreateReportUseCase) Execute(ctx context.Context, in CreateReportInput) (*CreateReportOutput, error) {
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
	uc.log.Info("report created", zap.String("reportId", report.ID), zap.String("processId", in.ProcessID))

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
		uc.log.Error("failed to publish report event",
			zap.String("event", event),
			zap.String("processId", processID),
			zap.Error(err),
		)
	}
}
