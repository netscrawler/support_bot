package pipeline

import (
	"context"
	"support_bot/internal/models"
)

type Runner interface {
	Run(
		ctx context.Context,
		step Step,
		data models.Dataset,
	) (models.Dataset, error)
}
