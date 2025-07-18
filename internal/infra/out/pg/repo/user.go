package pgrepo

import (
	"context"

	userrepo "support_bot/internal/infra/out/pg/gen/user"
	"support_bot/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	q *userrepo.Queries
}

func NewUser(s *pgx.Conn) *User {
	q := userrepo.New(s)
	return &User{
		q: q,
	}
}

func (u *User) Create(ctx context.Context, usr *models.User) error {
	username := pgtype.Text{}
	username.Scan(usr.Username)

	firstName := pgtype.Text{}
	firstName.Scan(usr.Username)

	lastName := pgtype.Text{}
	lastName.Scan(usr.Username)

	_, err := u.q.CreateUser(ctx, userrepo.CreateUserParams{
		TelegramID: usr.TelegramID,
		Username:   username,
		FirstName:  firstName,
		LastName:   lastName,
		Role:       userrepo.UserRole(usr.Role),
	})

	return err
}

func (u *User) Update(ctx context.Context, usr *models.User) error {
	username := pgtype.Text{}
	username.Scan(usr.Username)

	firstName := pgtype.Text{}
	firstName.Scan(usr.Username)

	lastName := pgtype.Text{}
	lastName.Scan(usr.Username)

	err := u.q.UpdateUser(ctx, userrepo.UpdateUserParams{
		Username:   username,
		TelegramID: usr.TelegramID,
		FirstName:  firstName,
		LastName:   lastName,
	})

	return err
}

func (u *User) GetByUsername(ctx context.Context, uname string) (*models.User, error) {
	username := pgtype.Text{}
	username.Scan(uname)

	user, err := u.q.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	retUser := userFromGenModel(user)

	return &retUser, nil
}

func (u *User) GetByTgID(ctx context.Context, id int64) (*models.User, error) {
	user, err := u.q.GetUserByTgID(ctx, id)
	if err != nil {
		return nil, err
	}

	retUser := userFromGenModel(user)
	return &retUser, nil
}

func (u *User) GetAllAdmins(ctx context.Context) ([]models.User, error) {
	admins, err := u.q.GetAllAdmins(ctx)
	if err != nil {
		return nil, err
	}

	adminList := make([]models.User, 0, len(admins))
	for _, u := range admins {
		adminList = append(adminList, userFromGenModel(u))
	}

	return adminList, nil
}

func (u *User) GetAll(ctx context.Context) ([]models.User, error) {
	users, err := u.q.GetAllAdmins(ctx)
	if err != nil {
		return nil, err
	}

	userList := make([]models.User, 0, len(users))
	for _, u := range users {
		userList = append(userList, userFromGenModel(u))
	}

	return userList, nil
}

func (u *User) Delete(ctx context.Context, tgID int64) error {
	err := u.q.DeleteUserbyTgID(ctx, tgID)
	return err
}

func userFromGenModel(u userrepo.User) models.User {
	return models.User{
		ID:         int(u.ID),
		TelegramID: u.TelegramID,
		Username:   u.Username.String,
		FirstName:  u.FirstName.String,
		LastName:   &u.FirstName.String,
		Role:       string(u.Role),
	}
}
