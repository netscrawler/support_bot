package orchestrator

import (
	"context"
	"errors"
	"log/slog"
	"support_bot/internal/models"
	"testing"
	"time"
)

// stubMsgDeleter — рукописная заглушка msgDeleter для тестов ветвления
// Deleter.deleteOne: возвращает заданную ошибку (или nil) и запоминает факт
// вызова. Мокери здесь не используется — у интерфейса один метод, заглушка
// проще и не требует правки .mockery.yaml ради непубличного интерфейса.
type stubMsgDeleter struct {
	err    error
	called bool
}

func (s *stubMsgDeleter) DeleteMsg(context.Context, models.SentMessage) error {
	s.called = true

	return s.err
}

// TestDeleter_DeleteOne проверяет выбор адаптера доставки по ChType/полям
// сообщения и решение о пометке "удалено" — единственную бизнес-логику,
// которую содержит Deleter (остальное — тонкие обёртки над repo).
func TestDeleter_DeleteOne(t *testing.T) {
	tests := []struct {
		name            string
		msg             models.SentMessage
		tgErr           error
		maxErr          error
		wantMarkDeleted bool
		wantTgCalled    bool
		wantMaxCalled   bool
	}{
		{
			name:            "успешная отправка через tg помечает сообщение удалённым",
			msg:             models.SentMessage{ChType: models.ChatTypeTg},
			wantMarkDeleted: true,
			wantTgCalled:    true,
		},
		{
			name:            "успешная отправка через max помечает сообщение удалённым",
			msg:             models.SentMessage{ChType: models.ChatTypeMax},
			wantMarkDeleted: true,
			wantMaxCalled:   true,
		},
		{
			name:            "неизвестный ChType с ненулевым MessageID маршрутизируется в tg",
			msg:             models.SentMessage{ChType: "", MessageID: 1},
			wantMarkDeleted: true,
			wantTgCalled:    true,
		},
		{
			name: "неизвестный ChType с MessageIDStr маршрутизируется в max, даже если задан и MessageID",
			msg: models.SentMessage{
				ChType:       "",
				MessageID:    1,
				MessageIDStr: new("mid"),
			},
			wantMarkDeleted: true,
			wantMaxCalled:   true,
		},
		{
			name:            "свежая ошибка удаления не помечает сообщение удалённым",
			msg:             models.SentMessage{ChType: models.ChatTypeTg, Time: time.Now()},
			tgErr:           errors.New("telegram недоступен"),
			wantMarkDeleted: false,
			wantTgCalled:    true,
		},
		{
			name: "ошибка удаления старше 48 часов всё равно помечает сообщение удалённым",
			msg: models.SentMessage{
				ChType: models.ChatTypeTg,
				Time:   time.Now().Add(-49 * time.Hour),
			},
			tgErr:           errors.New("telegram недоступен"),
			wantMarkDeleted: true,
			wantTgCalled:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tg := &stubMsgDeleter{err: tt.tgErr}
			mx := &stubMsgDeleter{err: tt.maxErr}
			d := &Deleter{tgDel: tg, maxDel: mx, log: slog.New(slog.DiscardHandler)}

			got := d.deleteOne(context.Background(), tt.msg)

			if got != tt.wantMarkDeleted {
				t.Errorf("deleteOne() = %v, want %v", got, tt.wantMarkDeleted)
			}
			if tg.called != tt.wantTgCalled {
				t.Errorf("tgDel.called = %v, want %v", tg.called, tt.wantTgCalled)
			}
			if mx.called != tt.wantMaxCalled {
				t.Errorf("maxDel.called = %v, want %v", mx.called, tt.wantMaxCalled)
			}
		})
	}
}
