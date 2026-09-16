package store

import (
	"context"
	"fmt"
	"log/slog"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserStore реализует service.UserProvider поверх sqlc-запросов user.sql.
// Заменяет internal/tg_bot/repository.UserRepository, чья GetByUsername
// содержала баг: nil-указатель передавался в sqlx.GetContext и приводил к
// панике при любом совпадении. Генерируемый sqlc-метод возвращает строку
// значением, поэтому такой класс ошибок здесь структурно невозможен.
type UserStore struct {
	q   *sqlcgen.Queries
	log *slog.Logger
}

func NewUserStore(pool *pgxpool.Pool, log *slog.Logger) *UserStore {
	return &UserStore{
		q:   sqlcgen.New(pool),
		log: log.With(slog.Any("module", "store.user")),
	}
}

func (s *UserStore) Create(ctx context.Context, user *models.User) error {
	_, err := s.q.CreateUser(ctx, sqlcgen.CreateUserParams{
		TelegramID: user.TelegramID,
		Username:   &user.Username,
		FirstName:  &user.FirstName,
		LastName:   user.LastName,
		Role:       sqlcgen.UserRole(user.Role),
	})
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

// Update намеренно не меняет role — так делал и старый UPDATE-запрос
// (WHERE username = ..., SET telegram_id/first_name/last_name без role).
func (s *UserStore) Update(ctx context.Context, user *models.User) error {
	if err := s.q.UpdateUser(ctx, sqlcgen.UpdateUserParams{
		Username:   &user.Username,
		TelegramID: user.TelegramID,
		FirstName:  &user.FirstName,
		LastName:   user.LastName,
	}); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

func (s *UserStore) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	row, err := s.q.GetUserByUsername(ctx, &username)
	if err != nil {
		return nil, translateNoRows(err)
	}

	user := mapUserRow(row.ID, row.TelegramID, row.Username, row.FirstName, row.LastName, row.Role)

	return &user, nil
}

func (s *UserStore) GetAll(ctx context.Context) ([]models.User, error) {
	rows, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]models.User, 0, len(rows))
	for _, row := range rows {
		users = append(
			users,
			mapUserRow(row.ID, row.TelegramID, row.Username, row.FirstName, row.LastName, row.Role),
		)
	}

	return users, nil
}

func (s *UserStore) GetByTgID(ctx context.Context, id int64) (*models.User, error) {
	row, err := s.q.GetUserByTelegramID(ctx, id)
	if err != nil {
		return nil, translateNoRows(err)
	}

	user := mapUserRow(row.ID, row.TelegramID, row.Username, row.FirstName, row.LastName, row.Role)

	return &user, nil
}

// GetAllAdmins возвращает пользователей с ролью admin/primary — тот же
// фильтр, что и в старом запросе (role in ('admin', 'primary')). Метод
// нужен новому service.AdminNotifier.
func (s *UserStore) GetAllAdmins(ctx context.Context) ([]models.User, error) {
	rows, err := s.q.ListAdmins(ctx)
	if err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}

	users := make([]models.User, 0, len(rows))
	for _, row := range rows {
		users = append(
			users,
			mapUserRow(row.ID, row.TelegramID, row.Username, row.FirstName, row.LastName, row.Role),
		)
	}

	return users, nil
}

func (s *UserStore) Delete(ctx context.Context, tgID int64) error {
	if err := s.q.DeleteUser(ctx, tgID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}

func mapUserRow(
	id int32,
	telegramID int64,
	username, firstName, lastName *string,
	role sqlcgen.UserRole,
) models.User {
	return models.User{
		ID:         int(id),
		TelegramID: telegramID,
		Username:   derefStr(username),
		FirstName:  derefStr(firstName),
		LastName:   lastName,
		Role:       string(role),
	}
}
