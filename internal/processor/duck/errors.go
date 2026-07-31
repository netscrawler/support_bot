package duck

import "errors"

var ErrCreateTableNoColumnInData = errors.New("no column in data to create table")
