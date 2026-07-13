package models

type ReportForTgLK struct {
	ID    int
	Name  string
	Title string
}

type LoadReportRPL struct {
	ReportsTotal int
	PageCount    int
	CurrentPage  int
	Reports      []ReportForTgLK
}
