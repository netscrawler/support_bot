package common

import "github.com/google/uuid"

type PathValue = string

const ReportIDPV PathValue = "report_id"

func ValidateReportIDPV(pv string) error {
	_, err := uuid.Parse(pv)
	if err != nil {
		return err
	}
	return nil
}
