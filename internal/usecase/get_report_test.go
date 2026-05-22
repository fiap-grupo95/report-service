package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/fiap/secure-systems/report-service/internal/domain"
	"github.com/google/uuid"
)

func TestGetReportExecute_InvalidID(t *testing.T) {
	uc := NewGetReportUseCase(&fakeReportRepository{})

	report, err := uc.Execute(context.Background(), "not-a-uuid")

	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
	if report != nil {
		t.Errorf("expected nil report, got %#v", report)
	}
}

func TestGetReportExecute_ReturnsReport(t *testing.T) {
	id := uuid.NewString()
	expected := &domain.Report{ID: id, ProcessID: "process-1"}
	repo := &fakeReportRepository{
		findByIDFunc: func(_ context.Context, gotID string) (*domain.Report, error) {
			if gotID != id {
				t.Errorf("expected id %s, got %s", id, gotID)
			}
			return expected, nil
		},
	}

	uc := NewGetReportUseCase(repo)
	report, err := uc.Execute(context.Background(), id)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report != expected {
		t.Errorf("expected repository report, got %#v", report)
	}
}

func TestGetReportExecute_RepositoryErrorIsWrapped(t *testing.T) {
	id := uuid.NewString()
	repoErr := domain.ErrReportNotFound
	repo := &fakeReportRepository{
		findByIDFunc: func(_ context.Context, _ string) (*domain.Report, error) {
			return nil, repoErr
		},
	}

	uc := NewGetReportUseCase(repo)
	report, err := uc.Execute(context.Background(), id)

	if !errors.Is(err, repoErr) {
		t.Fatalf("expected wrapped repo error, got %v", err)
	}
	if report != nil {
		t.Errorf("expected nil report, got %#v", report)
	}
}
