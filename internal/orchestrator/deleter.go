package orchestrator

import (
	"context"
	"log/slog"
	"support_bot/internal/models"
	"time"
)

type msgDeleter interface {
	DeleteMsg(ctx context.Context, msg models.SentMessage) error
}

// deleterStore — контракт, который Deleter требует от хранилища сообщений.
// Выделен в интерфейс вместо конкретного SentMsgRepository (как было до
// sqlc-миграции), поскольку реализация теперь живёт в другом пакете
// (internal/store) и обращаться к её методам как к пакетно-приватным больше
// нельзя; заодно делает ветвление в Deleter.delete тестируемым без БД.
type deleterStore interface {
	WithLockedMsgsToDelete(
		ctx context.Context,
		fn func(ctx context.Context, msg models.SentMessage) (markDeleted bool),
	) error
	RemoveDeletedMessages(ctx context.Context) (int64, error)
	MarkEndOfDayMsgDeleted(ctx context.Context) error
}

type Deleter struct {
	tgDel  msgDeleter
	maxDel msgDeleter

	repo deleterStore

	evC chan models.Event

	log *slog.Logger
}

func NewDeleter(
	evC chan models.Event,
	tgDel msgDeleter,
	maxDel msgDeleter,
	repo deleterStore,
	log *slog.Logger,
) *Deleter {
	l := log.With(slog.Any("module", "deleter"))

	return &Deleter{
		tgDel:  tgDel,
		maxDel: maxDel,
		repo:   repo,
		log:    l,
		evC:    evC,
	}
}

func (d *Deleter) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				d.log.InfoContext(
					ctx,
					"context canceled deleter stopped",
					slog.Any("err", ctx.Err()),
				)

				return
			case e, ok := <-d.evC:
				if !ok {
					d.log.InfoContext(ctx, "event chan closed")

					return
				}

				d.log.InfoContext(ctx, "receiving event", slog.Any("event", e))

				if e.Type != models.EventTypeDeleteSentReport {
					d.log.ErrorContext(ctx, "unexpected event type", slog.Any("event", e))
				}

				d.delete(ctx)
			}
		}
	}()
}

func (d *Deleter) delete(ctx context.Context) {
	d.markLastMsgAsDeletedWithoutDelete(ctx)
	d.clearDeletedMessages(ctx)
	d.log.InfoContext(ctx, "start deleting messages")

	err := d.repo.WithLockedMsgsToDelete(ctx, d.deleteOne)
	if err != nil {
		d.log.ErrorContext(ctx, "failed to process messages to delete", slog.Any("err", err))
	}
}

// deleteOne выбирает адаптер доставки по типу чата и решает, помечать ли
// сообщение удалённым — то же ветвление, что было инлайново в старом
// Deleter.delete: для "неизвестного" ChType сначала пробуем tg по
// ненулевому MessageID, затем max по непустому MessageIDStr (max выигрывает,
// если заданы оба — так вело себя и старое ветвление, MessageIDStr
// проверяется после MessageID и перезаписывает выбор). Сообщение помечается
// удалённым при успехе или если ошибка удаления держится ≥48 часов (чтобы не
// пытаться бесконечно); иначе остаётся залоченным до следующего тика.
func (d *Deleter) deleteOne(ctx context.Context, m models.SentMessage) bool {
	var del msgDeleter

	switch m.ChType {
	case models.ChatTypeMax:
		del = d.maxDel
	case models.ChatTypeTg:
		del = d.tgDel
	default:
		if m.MessageID != 0 {
			del = d.tgDel
		}

		if m.MessageIDStr != nil {
			del = d.maxDel
		}
	}

	if err := del.DeleteMsg(ctx, m); err != nil {
		d.log.ErrorContext(ctx, "failed to delete messages", slog.Any("err", err))

		return time.Since(m.Time) >= 48*time.Hour
	}

	return true
}

func (d *Deleter) clearDeletedMessages(ctx context.Context) {
	d.log.InfoContext(ctx, "begin clear deleted messages")

	removed, err := d.repo.RemoveDeletedMessages(ctx)
	if err != nil {
		d.log.ErrorContext(ctx, "failed to clear deleted messages", slog.Any("err", err))
	}

	d.log.InfoContext(ctx, "end clear deleted messages", slog.Any("removed", removed))
}

func (d *Deleter) markLastMsgAsDeletedWithoutDelete(ctx context.Context) {
	d.log.InfoContext(ctx, "begin mark last msg as deleted")

	err := d.repo.MarkEndOfDayMsgDeleted(ctx)
	if err != nil {
		d.log.ErrorContext(ctx, "mark last messages as deleted error", slog.Any("error", err))
	}

	d.log.InfoContext(ctx, "end mark last msg as deleted")
}
