package metrics

import "context"

type reportKey struct{}

// WithReport кладёт имя отчёта в ctx, чтобы нижние слои ставили его в label report.
func WithReport(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, reportKey{}, name)
}

// ReportFrom возвращает имя отчёта из ctx или "".
func ReportFrom(ctx context.Context) string {
	if name, ok := ctx.Value(reportKey{}).(string); ok {
		return name
	}

	return ""
}
