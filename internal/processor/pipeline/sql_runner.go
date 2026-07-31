package pipeline

import (
	"context"
	"support_bot/internal/models"

	"support_bot/internal/processor/duck"
)

type SqlRunner struct{}

func (s *SqlRunner) Run(
	ctx context.Context,
	step Step,
	data models.Dataset,
) (models.Dataset, error) {
	db, err := duck.New()
	defer db.Close()
	if err != nil {
		return nil, err
	}

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
