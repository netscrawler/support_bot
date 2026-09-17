package service

import (
	"context"
	"errors"
	"log/slog"
	"support_bot/internal/models"
	"testing"
)

// fakeChatProvider — рукописная реализация ChatProvider для тестов Chat.Add.
// GetByTitle паникует по умолчанию: этот тест-план проверяет, что Add больше
// не обращается к GetByTitle для проверки дубликатов (решение об уникальности
// теперь принимается по chat_id через Exists).
type fakeChatProvider struct {
	existsFn func(ctx context.Context, chatID int64) (bool, error)

	createCalled bool
	createChat   *models.TgChatDTO
	createErr    error
}

func (f *fakeChatProvider) Create(_ context.Context, chat *models.TgChatDTO) error {
	f.createCalled = true
	f.createChat = chat

	return f.createErr
}

func (f *fakeChatProvider) Exists(ctx context.Context, chatID int64) (bool, error) {
	return f.existsFn(ctx, chatID)
}

func (f *fakeChatProvider) GetByTitle(context.Context, string) (*models.TgChatDTO, error) {
	panic("GetByTitle must not be called by Chat.Add: duplicates are now identified by chat_id")
}

func (f *fakeChatProvider) GetAll(context.Context) ([]models.TgChatDTO, error) {
	panic("GetAll is not exercised by these tests")
}

func (f *fakeChatProvider) Delete(context.Context, int64) error {
	panic("Delete is not exercised by these tests")
}

// fakeAdminNotifier не возвращает ни одного администратора, поэтому
// Notify.SendAdminNotify завершается до обращения к Telegram — это позволяет
// сконструировать реальный *Notify в тесте без production-сеама для подмены
// доставки.
type fakeAdminNotifier struct{}

func (fakeAdminNotifier) GetAllAdmins(context.Context) ([]models.User, error) {
	return nil, nil
}

func newTestNotify() *Notify {
	return NewNotify(nil, fakeAdminNotifier{}, slog.Default())
}

// TestChat_Add_AllowsSameTitleWithDifferentChatID проверяет, что регистрация
// разрешена, если title совпадает с уже существующим чатом, но chat_id другой —
// это одобренное изменение поведения: раньше проверка на дубликат шла по
// title, что ошибочно блокировало разные Telegram-группы с одинаковым именем.
func TestChat_Add_AllowsSameTitleWithDifferentChatID(t *testing.T) {
	repo := &fakeChatProvider{
		existsFn: func(_ context.Context, chatID int64) (bool, error) {
			if chatID != 200 {
				t.Fatalf("Exists() called with chatID=%d, want 200", chatID)
			}

			return false, nil
		},
	}

	c := NewChat(repo, newTestNotify(), slog.Default())

	chat := &models.TgChatDTO{ChatID: 200, Title: "Same Title"}

	err := c.Add(context.Background(), chat)
	if err != nil {
		t.Fatalf("Add() error = %v, want nil", err)
	}
	if !repo.createCalled {
		t.Error("Create() was not called, want it to be called")
	}
	if repo.createChat.ChatID != 200 {
		t.Errorf("Create() called with ChatID=%d, want 200", repo.createChat.ChatID)
	}
}

// TestChat_Add_RejectsDuplicateChatID проверяет, что регистрация отклоняется,
// когда chat_id уже существует — это единственный случай, который теперь
// считается дубликатом.
func TestChat_Add_RejectsDuplicateChatID(t *testing.T) {
	repo := &fakeChatProvider{
		existsFn: func(_ context.Context, chatID int64) (bool, error) {
			if chatID != 100 {
				t.Fatalf("Exists() called with chatID=%d, want 100", chatID)
			}

			return true, nil
		},
	}

	c := NewChat(repo, newTestNotify(), slog.Default())

	chat := &models.TgChatDTO{ChatID: 100, Title: "Existing Chat"}

	err := c.Add(context.Background(), chat)
	if !errors.Is(err, models.ErrAlreadyExist) {
		t.Fatalf("Add() error = %v, want errors.Is(err, models.ErrAlreadyExist)", err)
	}
	if repo.createCalled {
		t.Error("Create() was called, want it not to be called")
	}
}

// TestChat_Add_LookupErrorNotifiesAndDoesNotCreate проверяет, что ошибка
// проверки существования чата приводит к отказу (fail closed): Create не
// вызывается, ошибка одновременно относится к models.ErrInternal и к
// исходной ошибке БД, а уведомление администраторам отправляется без паники
// (список администраторов пуст).
func TestChat_Add_LookupErrorNotifiesAndDoesNotCreate(t *testing.T) {
	dbErr := errors.New("connection reset")

	repo := &fakeChatProvider{
		existsFn: func(context.Context, int64) (bool, error) {
			return false, dbErr
		},
	}

	c := NewChat(repo, newTestNotify(), slog.Default())

	chat := &models.TgChatDTO{ChatID: 300, Title: "Some Chat"}

	err := c.Add(context.Background(), chat)
	if !errors.Is(err, models.ErrInternal) {
		t.Errorf("Add() error = %v, want errors.Is(err, models.ErrInternal)", err)
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("Add() error = %v, want errors.Is(err, dbErr)", err)
	}
	if repo.createCalled {
		t.Error("Create() was called, want it not to be called")
	}
}
