package errorz

import "fmt"

type ClientErr struct {
	Code int
	Desc string
}

func (c *ClientErr) Error() string {
	return fmt.Sprintf("Error: %s", c.Desc)
}

func (c *ClientErr) Unwrap() error {
	return fmt.Errorf("%s", c.Desc)
}
