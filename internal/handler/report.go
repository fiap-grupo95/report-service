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

	r, err := h.uc.Execute(c.Request.Context(), reportID)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
			return
		}
		if errors.Is(err, domain.ErrReportNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
			return
		}
		logging.LoggerWithContext(c.Request.Context()).Error().
			Str("report_id", reportID).Err(err).Msg("get report failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
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
