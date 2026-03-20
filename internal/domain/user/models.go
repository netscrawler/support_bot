package user

type UserDBO struct {
	ID       string `db:"id"`
	Login    string `db:"login"`
	Email    string `db:"email"`
	Role     string `db:"role"`
	Password string `db:"password"`
}
