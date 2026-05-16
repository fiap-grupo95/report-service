package usecase

import (
	"context"
	"fmt"

	"github.com/fiap/secure-systems/report-service/internal/domain"
	"github.com/google/uuid"
)

type GetReportUseCase struct {
	repo ReportRepository
}

func NewGetReportUseCase(repo ReportRepository) *GetReportUseCase {
	return &GetReportUseCase{repo: repo}
}

func (uc *GetReportUseCase) Execute(ctx context.Context, reportID string) (*domain.Report, error) {
	if _, err := uuid.Parse(reportID); err != nil {
		return nil, domain.ErrInvalidID
	}

	r, err := uc.repo.FindByID(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("find report: %w", err)
	}
	return r, nil
}
