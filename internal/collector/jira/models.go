package jira

import "support_bot/internal/pkg/funcs"

const MetaKey = "_meta"

type SearchRPL struct {
	Expand     string           `json:"expand"`
	StartAt    int              `json:"startAt"`
	MaxResults int              `json:"maxResults"`
	Total      int              `json:"total"`
	Issues     []map[string]any `json:"issues"`
}

func (f *SearchRPL) Flatten() {
	for i := range f.Issues {
		f.Issues[i] = funcs.Flatten(f.Issues[i], "fields")
	}
}

func (f *SearchRPL) GetMap() []map[string]any {
	result := make([]map[string]any, 0, len(f.Issues)+1)

	// первая запись — метаданные
	result = append(result, map[string]any{
		MetaKey: map[string]any{
			"total":      f.Total,
			"startAt":    f.StartAt,
			"maxResults": f.MaxResults,
		},
	})

	return result
}
