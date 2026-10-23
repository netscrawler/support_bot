package pipeline

import (
	"context"

	"support_bot/internal/models"
	"support_bot/internal/processor/duck"
)

type SQLRunner struct{}

func (s *SQLRunner) Run(
	ctx context.Context,
	step Step,
	data models.Dataset,
) (models.Dataset, error) {
	db, err := duck.New()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	err = db.LoadDataFromMapSlice(ctx, data)
	if err != nil {
		return nil, err
	}

	out, err := db.ExecuteQuery(ctx, step.Script)
	if err != nil {
		return nil, err
	}

	var outKey string

	_, ok := data[step.ID]
	if ok {
		outKey = step.ID + "_duck"
	} else {
		outKey = step.ID
	}

	data[outKey] = out

	return data, nil
}
