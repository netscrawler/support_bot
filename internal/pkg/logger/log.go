package logger

import (
	"context"
	"log/slog"
)

type ctxKey string

const (
	slogFields ctxKey = "slog_fields"
	TraceField        = "trace_id"
)

type ContextHandler struct {
	slog.Handler
}

// Handle ....
func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(slogFields).([]slog.Attr); ok {
		for _, v := range attrs {
			r.AddAttrs(v)
		}
	}

	return h.Handler.Handle(ctx, r)
}

func (h ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{
		h.Handler.WithAttrs(attrs),
	}
}

func (h ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{
		Handler: h.Handler.WithGroup(name),
	}
}

func (h ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Handler.Enabled(ctx, level)
}

// AppendCtx included in any Record created with such context.
func AppendCtx(parent context.Context, attr ...slog.Attr) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(slogFields).([]slog.Attr); ok {
		newAttrs := make([]slog.Attr, 0, len(v)+len(attr))
		newAttrs = append(newAttrs, v...)
		newAttrs = append(newAttrs, attr...)

		return context.WithValue(parent, slogFields, newAttrs)
	}

	v := []slog.Attr{}
	v = append(v, attr...)

	return context.WithValue(parent, slogFields, v)
}

func WithTraceID(ctx context.Context, id string) context.Context {
	return AppendCtx(ctx, slog.String(TraceField, id))
}

func GetTraceID(ctx context.Context) string {
	if attrs, ok := ctx.Value(slogFields).([]slog.Attr); ok {
		for _, a := range attrs {
			if a.Key == TraceField {
				if v, ok := a.Value.Any().(string); ok {
					return v
				}
			}
		}
	}

	return ""
}
