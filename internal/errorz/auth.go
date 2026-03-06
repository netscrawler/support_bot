package errorz

type AuthErr struct {
	Desc string
	Err  error
}

func NewAuthErr(desc string, err error) *AuthErr {
	return &AuthErr{
		Desc: desc,
		Err:  err,
	}
}

func (a AuthErr) Error() string {
	if a.Desc != "" {
		return a.Desc
	}
	return a.Err.Error()
}
