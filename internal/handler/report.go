package handler

import (
	"errors"
	"net/http"

	"github.com/fiap/secure-systems/report-service/internal/domain"
	"github.com/fiap/secure-systems/report-service/internal/logging"
	"github.com/fiap/secure-systems/report-service/internal/usecase"
	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	uc *usecase.GetReportUseCase
}

func NewReportHandler(uc *usecase.GetReportUseCase) *ReportHandler {
	return &ReportHandler{uc: uc}
}

// GET /internal/reports/:reportId
func (h *ReportHandler) GetReport(c *gin.Context) {
	reportID := c.Param("reportId")
	log := logging.LoggerWithContext(c.Request.Context())

	r, err := h.uc.Execute(c.Request.Context(), reportID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidID):
			log.Warn().
				Str("report_id", reportID).
				Msg("get report rejected: invalid report id format")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		case errors.Is(err, domain.ErrReportNotFound):
			log.Warn().
				Str("report_id", reportID).
				Msg("get report: report not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		default:
			log.Error().
				Err(err).
				Str("report_id", reportID).
				Msg("get report failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"report_id":       r.ID,
		"process_id":      r.ProcessID,
		"components":      r.Analysis.Components,
		"risks":           r.Analysis.Risks,
		"recommendations": r.Analysis.Recommendations,
		"created_at":      r.CreatedAt,
	})
}
