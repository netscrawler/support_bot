package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"support_bot/internal/api/http/common"
	"support_bot/internal/models"
	"support_bot/internal/pkg/httplib"
)

// /api/v1/public/report/{report_id}
func (h *Handler) GetGeneratedReportByID(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue(common.ReportIDPV)

	err := common.ValidateReportIDPV(reportID)
	if err != nil {
		httplib.NewHTTPError(http.StatusBadRequest, err.Error()).Write(w)
		return
	}

	report, err := h.rp.GenerateReport(r.Context(), reportID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			httplib.ErrNotFound.Write(w)
			return
		}

		h.log.ErrorContext(r.Context(), "generate report failed", slog.Any("error", err))
		httplib.ErrInternal.Write(w)
		return
	}

	httplib.File(w, report.FileName, report.Data)
}
