package handler

import (
	"errors"
	"net/http"

	"github.com/fiap/secure-systems/report-service/internal/domain"
	"github.com/fiap/secure-systems/report-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ReportHandler struct {
	uc  *usecase.GetReportUseCase
	log *zap.Logger
}

func NewReportHandler(uc *usecase.GetReportUseCase, log *zap.Logger) *ReportHandler {
	return &ReportHandler{uc: uc, log: log}
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
		h.log.Error("get report failed", zap.String("reportId", reportID), zap.Error(err))
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
