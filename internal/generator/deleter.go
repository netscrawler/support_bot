package generator

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	models "support_bot/internal/models/report"
	"time"
)

type tgMsgDeleter interface {
	DeleteMsg(msg models.TgMessage) error
}

type Deleter struct {
	tgDel tgMsgDeleter

	repo SentMsgRepository

	evC chan models.Event

	log *slog.Logger
}

func NewDeleter(evC chan models.Event, tgDel tgMsgDeleter, repo SentMsgRepository, log *slog.Logger) *Deleter {
	l := log.With(slog.Any("module", "deleter"))
	return &Deleter{
		tgDel: tgDel,
		repo:  repo,
		log:   l,
		evC:   evC,
	}
}

func (d *Deleter) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				d.log.InfoContext(ctx, "context canceled deleter stopped", slog.Any("err", ctx.Err()))
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

	msg, u, err := d.repo.LoadMsgToDelete(ctx)
	if err != nil {
		d.log.ErrorContext(ctx, "failed to load messages", slog.Any("err", err))
		return
	}
	defer func() {
		err = u.Rollback()
		if err != nil {
			if !errors.Is(err, sql.ErrTxDone) {
				d.log.ErrorContext(ctx, "failed to rollback messages", slog.Any("err", err))
			}
		}
	}()

	for _, m := range msg {
		err := d.tgDel.DeleteMsg(m)
		if err != nil {
			d.log.ErrorContext(ctx, "failed to delete messages", slog.Any("err", err))
			if time.Since(m.Time) < 72*time.Hour {
				continue
			}
		}

		err = d.repo.MarkDeleted(ctx, m.ID, u)
		if err != nil {
			d.log.ErrorContext(ctx, "failed to mark messages as deleted", slog.Any("err", err))
			continue
		}

	}

	err = u.Commit()
	if err != nil {
		d.log.ErrorContext(ctx, "failed to commit messages as deleted", slog.Any("err", err))
	}
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
