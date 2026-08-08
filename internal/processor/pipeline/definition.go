package pipeline

import "support_bot/internal/models"

// type Pipeline struct {
//	Name  string `json:"name"`
//	Steps []Step `json:"steps"`
//}

type (
	Pipeline = models.Pipeline
	Step     = models.Step
)

// type Step struct {
//	ID     string         `json:"id,omitempty"`
//	Type   string         `json:"type"`
//	Script string         `json:"script,omitempty"`
//	Params map[string]any `json:"params,omitempty"`
//}
