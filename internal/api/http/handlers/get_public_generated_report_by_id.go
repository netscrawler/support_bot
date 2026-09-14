package handlers

import (
	"errors"
	"net/http"
	"support_bot/internal/api/http/common"
	"support_bot/internal/errorz"
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

	report, err := h.rp.GenerateReport(reportID)
	if err != nil {
		if errors.Is(err, errorz.ErrNotFound) {
			httplib.ErrNotFound.Write(w)
			return
		}

		httplib.NewHTTPError(http.StatusInternalServerError, err.Error()).Write(w)
		return
	}

	w.WriteHeader(http.StatusOK)
	httplib.File(w, report.FileName, report.Data)
}
