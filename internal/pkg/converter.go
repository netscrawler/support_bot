package pkg

import "sort"

func ConvertSortedRows(
	rows []map[string]any,
	columns []string,
) [][]any {
	if len(rows) == 0 {
		return nil
	}

	cols := columns

	if len(cols) == 0 {
		cols = make([]string, 0, len(rows[0]))
		for k := range rows[0] {
			cols = append(cols, k)
		}

		sort.Strings(cols)
	}

	matrix := make([][]any, 0, len(rows)+1)

	header := make([]any, len(cols))
	for i, col := range cols {
		header[i] = col
	}

	matrix = append(matrix, header)

	for _, rowMap := range rows {
		row := make([]any, len(cols))
		for i, col := range cols {
			row[i] = rowMap[col]
		}

		matrix = append(matrix, row)
	}

	return matrix
}
