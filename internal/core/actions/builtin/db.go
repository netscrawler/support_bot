package builtin

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

type DB struct {
	db *sql.DB

	schema map[string]map[string]string
}

func NewDB() (DB, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return DB{}, fmt.Errorf("error opening db: %w", err)
	}

	return DB{db: db, schema: make(map[string]map[string]string)}, nil
}

func (d *DB) LoadDataFromMapSlice(ctx context.Context, sample map[string][]map[string]any) error {
	for table, data := range sample {
		err := d.createTableFromMap(ctx, table, data)
		if err != nil {
			return fmt.Errorf("error creating table %s: %w", table, err)
		}

		err = d.InsertData(ctx, table, data)
		if err != nil {
			return fmt.Errorf("error inserting data into table %s: %w", table, err)
		}
	}

	return nil
}

func (d *DB) ExecuteQuery(ctx context.Context, query string) ([]map[string]any, error) {
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error executing query %s: %w", query, err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("error getting columns: %w", err)
	}

	var results []map[string]any

	for rows.Next() {
		values := make([]any, len(columns))

		valuePtrs := make([]any, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		rowMap := make(map[string]any)

		for i, col := range columns {
			val := values[i]

			rowMap[col] = val
		}

		results = append(results, rowMap)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) InsertData(ctx context.Context, table string, rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}

	columns := make([]string, 0, len(rows[0]))

	for col := range d.schema[table] {
		columns = append(columns, col)
	}

	sort.Strings(columns)

	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	placeholderStr := "(" + strings.Join(placeholders, ", ") + ")"

	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES ",
		table,
		strings.Join(columns, ", "),
	)

	values := make([]any, 0, len(rows)*len(columns))
	placeholderRows := make([]string, len(rows))

	for i, row := range rows {
		for _, col := range columns {
			values = append(values, row[col])
		}

		placeholderRows[i] = placeholderStr
	}

	insertSQL += strings.Join(placeholderRows, ", ")

	_, err := d.db.ExecContext(ctx, insertSQL, values...)

	return err
}

func (d *DB) createTableFromMap(ctx context.Context, table string, sample []map[string]any) error {
	colDefs := make([]string, 0, len(sample))
	schema := make(map[string]string)

	for _, record := range sample {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		for name, val := range record {
			if _, ok := schema[name]; ok {
				continue
			}

			t, err := duckType(val)
			if err != nil {
				return err
			}

			schema[name] = t
			colDefs = append(colDefs, fmt.Sprintf("%s %s", name, t))
		}
	}

	query := fmt.Sprintf(
		"CREATE TABLE %s (%s)",
		table,
		strings.Join(colDefs, ","),
	)

	_, err := d.db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating table %s: %w", table, err)
	}

	d.schema[table] = schema

	return nil
}

func duckType(v any) (string, error) {
	switch v.(type) {
	case int, int32:
		return "INTEGER", nil
	case int64:
		return "BIGINT", nil
	case float32:
		return "FLOAT", nil
	case float64:
		return "DOUBLE", nil
	case bool:
		return "BOOLEAN", nil
	case string:
		return "VARCHAR", nil
	case time.Time:
		return "TIMESTAMP", nil
	case []byte:
		return "BLOB", nil
	case nil:
		return "VARCHAR", nil
	default:
		return "", fmt.Errorf("unsupported type: %T", v)
	}
}
