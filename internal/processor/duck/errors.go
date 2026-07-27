package duck

import "errors"

var ErrCreateTableNoColumnInData = errors.New("duckdb: no column in data to create table")
