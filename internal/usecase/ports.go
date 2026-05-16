package usecase

import (
	"context"

	"github.com/fiap/secure-systems/report-service/internal/domain"
)

type ReportRepository interface {
	Save(ctx context.Context, r *domain.Report) error
	FindByID(ctx context.Context, id string) (*domain.Report, error)
}

type EventPublisher interface {
	PublishToExchange(ctx context.Context, exchange string, payload []byte) error
}
