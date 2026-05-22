package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fiap/secure-systems/report-service/internal/domain"
	"github.com/fiap/secure-systems/report-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeReportRepository struct {
	findByIDFunc func(ctx context.Context, id string) (*domain.Report, error)
}

func (f *fakeReportRepository) Save(context.Context, *domain.Report) error {
	return nil
}

func (f *fakeReportRepository) FindByID(ctx context.Context, id string) (*domain.Report, error) {
	if f.findByIDFunc != nil {
		return f.findByIDFunc(ctx, id)
	}
	return nil, domain.ErrReportNotFound
}

func setupRouter(repo usecase.ReportRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewReportHandler(usecase.NewGetReportUseCase(repo))
	r.GET("/internal/reports/:reportId", h.GetReport)
	return r
}

func TestGetReport_ReturnsReport(t *testing.T) {
	reportID := uuid.NewString()
	createdAt := time.Date(2026, 5, 21, 10, 30, 0, 0, time.UTC)
	repo := &fakeReportRepository{
		findByIDFunc: func(_ context.Context, id string) (*domain.Report, error) {
			if id != reportID {
				t.Errorf("expected report id %s, got %s", reportID, id)
			}
			return &domain.Report{
				ID:        reportID,
				ProcessID: "process-1",
				Analysis: domain.Analysis{
					Components:      []string{"api"},
					Risks:           []string{"public bucket"},
					Recommendations: []string{"restrict access"},
				},
				CreatedAt: createdAt,
			}, nil
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/reports/"+reportID, nil)

	setupRouter(repo).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if body["report_id"] != reportID {
		t.Errorf("expected report_id %s, got %v", reportID, body["report_id"])
	}
	if body["process_id"] != "process-1" {
		t.Errorf("expected process_id process-1, got %v", body["process_id"])
	}
}

func TestGetReport_InvalidID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/reports/not-a-uuid", nil)

	setupRouter(&fakeReportRepository{}).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	assertErrorResponse(t, w, "invalid report id")
}

func TestGetReport_NotFound(t *testing.T) {
	reportID := uuid.NewString()
	repo := &fakeReportRepository{
		findByIDFunc: func(context.Context, string) (*domain.Report, error) {
			return nil, domain.ErrReportNotFound
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/reports/"+reportID, nil)

	setupRouter(repo).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
	assertErrorResponse(t, w, "report not found")
}

func TestGetReport_InternalError(t *testing.T) {
	reportID := uuid.NewString()
	repo := &fakeReportRepository{
		findByIDFunc: func(context.Context, string) (*domain.Report, error) {
			return nil, errors.New("mongo unavailable")
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/reports/"+reportID, nil)

	setupRouter(repo).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	assertErrorResponse(t, w, "internal error")
}

func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expected string) {
	t.Helper()

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if body["error"] != expected {
		t.Errorf("expected error %q, got %q", expected, body["error"])
	}
}
