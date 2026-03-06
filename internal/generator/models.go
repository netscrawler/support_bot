package generator

import "encoding/json"

type Workflow struct {
	Steps []Step
}

type Step struct {
	Name   string
	Action Action
}

const (
	STDAction    = "std"
	PluginAction = "plugin"
)

type Action string

const (
	ActionCollect  Action = "collect"
	ActionEvaluate Action = "evaluate"
	ActionExport   Action = "export"
	ActionSend     Action = "send"
)

const (
	DefaultWorkflow Action = "std@default"
	PluginCustom    Action = "plugin@custom"
)

func UnmarshallWorkflow(data []byte) (Workflow, error) {
	var w Workflow

	err := json.Unmarshal(data, &w)

	return w, err
}
