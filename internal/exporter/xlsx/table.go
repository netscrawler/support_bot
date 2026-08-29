package xlsx

// table-specific heuristics
const (
	rowsPerGridRow  = 6
	headerRowsCount = 1
)

type tableConfig struct {
	Range             string
	Name              string
	StyleName         string
	ShowColumnStripes bool
	ShowFirstColumn   bool
	ShowHeaderRow     *bool
	ShowLastColumn    bool
	ShowRowStripes    *bool
}
