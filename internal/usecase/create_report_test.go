package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/fiap/secure-systems/report-service/internal/domain"
	"go.uber.org/zap"
)

type fakeReportRepository struct {
	saveFunc     func(ctx context.Context, r *domain.Report) error
	findByIDFunc func(ctx context.Context, id string) (*domain.Report, error)
}

func (f *fakeReportRepository) Save(ctx context.Context, r *domain.Report) error {
	if f.saveFunc != nil {
		return f.saveFunc(ctx, r)
	}
	return nil
}

func (f *fakeReportRepository) FindByID(ctx context.Context, id string) (*domain.Report, error) {
	if f.findByIDFunc != nil {
		return f.findByIDFunc(ctx, id)
	}
	return nil, domain.ErrReportNotFound
}

type fakeEventPublisher struct {
	publishFunc func(ctx context.Context, exchange string, payload []byte) error
}

func (f *fakeEventPublisher) PublishToExchange(ctx context.Context, exchange string, payload []byte) error {
	if f.publishFunc != nil {
		return f.publishFunc(ctx, exchange, payload)
	}
	return nil
}

func TestCreateReportExecute_SavesReportAndPublishesCreatedEvent(t *testing.T) {
	var saved *domain.Report
	repo := &fakeReportRepository{
		saveFunc: func(_ context.Context, r *domain.Report) error {
			saved = r
			return nil
		},
	}

	var exchange string
	var event reportEvent
	publisher := &fakeEventPublisher{
		publishFunc: func(_ context.Context, ex string, payload []byte) error {
			exchange = ex
			if err := json.Unmarshal(payload, &event); err != nil {
				t.Fatalf("invalid event payload: %v", err)
			}
			return nil
		},
	}

	uc := NewCreateReportUseCase(repo, publisher, "report.topic", zap.NewNop())
	out, err := uc.Execute(context.Background(), CreateReportInput{
		ProcessID: "process-1",
		Analysis: domain.Analysis{
			Components:      []string{"api"},
			Risks:           []string{"public bucket"},
			Recommendations: []string{"restrict access"},
		},
		RawResponse: "raw",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("expected output")
	}
	if saved == nil {
		t.Fatal("expected report to be saved")
	}
	if saved.ID == "" || out.ReportID != saved.ID {
		t.Errorf("expected generated report id to be returned, got saved=%q output=%q", saved.ID, out.ReportID)
	}
	if saved.ProcessID != "process-1" {
		t.Errorf("expected process-1, got %s", saved.ProcessID)
	}
	if saved.CreatedAt.IsZero() || out.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if exchange != "report.topic" {
		t.Errorf("expected report.topic exchange, got %s", exchange)
	}
	if event.Event != "report_created" {
		t.Errorf("expected report_created event, got %s", event.Event)
	}
	if event.ReportID != saved.ID {
		t.Errorf("expected report id in event, got %s", event.ReportID)
	}
}

func TestCreateReportExecute_SaveErrorPublishesFailedEvent(t *testing.T) {
	repo := &fakeReportRepository{
		saveFunc: func(_ context.Context, _ *domain.Report) error {
			return errors.New("mongo unavailable")
		},
	}

	var event reportEvent
	publisher := &fakeEventPublisher{
		publishFunc: func(_ context.Context, _ string, payload []byte) error {
			if err := json.Unmarshal(payload, &event); err != nil {
				t.Fatalf("invalid event payload: %v", err)
			}
			return nil
		},
	}

	uc := NewCreateReportUseCase(repo, publisher, "report.topic", zap.NewNop())
	out, err := uc.Execute(context.Background(), CreateReportInput{ProcessID: "process-1"})

	if err == nil {
		t.Fatal("expected error")
	}
	if out != nil {
		t.Errorf("expected nil output, got %#v", out)
	}
	if event.Event != "report_failed" {
		t.Errorf("expected report_failed event, got %s", event.Event)
	}
	if event.ErrorMsg != "mongo unavailable" {
		t.Errorf("expected original error in event, got %q", event.ErrorMsg)
	}
}

func TestCreateReportExecute_PublishErrorDoesNotFailSuccessfulSave(t *testing.T) {
	uc := NewCreateReportUseCase(
		&fakeReportRepository{},
		&fakeEventPublisher{publishFunc: func(_ context.Context, _ string, _ []byte) error {
			return errors.New("rabbit unavailable")
		}},
		"report.topic",
		zap.NewNop(),
	)

	out, err := uc.Execute(context.Background(), CreateReportInput{ProcessID: "process-1"})
	if err != nil {
		t.Fatalf("expected publish failure to be logged only, got %v", err)
	}
	if out == nil || out.ReportID == "" {
		t.Fatalf("expected report output, got %#v", out)
	}
}
