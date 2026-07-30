// Package query runs arbitrary SQL against Snowflake and turns the result
// set into plain, JSON-safe Go values -- the core of snowstorm's
// data-access-first design: one execution path, callers choose how to
// render it (JSON today; table/CSV/etc. can render the same Result later).
package query

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// Result is the JSON-safe outcome of running one SQL statement.
type Result struct {
	Columns  []string         `json:"columns"`
	Rows     []map[string]any `json:"rows"`
	RowCount int              `json:"row_count"`
}

// Run executes sqlText and collects every row into a Result. Column values
// are normalized so the whole thing marshals cleanly with encoding/json:
//   - VARIANT/OBJECT/ARRAY columns are decoded from Snowflake's JSON-text
//     wire format into native JSON (object/array/number/etc.), not left as
//     a double-encoded string.
//   - BINARY columns are base64-encoded (raw bytes aren't valid JSON text).
//   - TIMESTAMP/DATE/TIME columns are formatted as RFC3339Nano strings.
//   - Everything else (numbers, strings, bools, NULL) passes through as-is.
func Run(ctx context.Context, db *sql.DB, sqlText string) (*Result, error) {
	rows, err := db.QueryContext(ctx, sqlText)
	if err != nil {
		return nil, fmt.Errorf("query: execute: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("query: read columns: %w", err)
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("query: read column types: %w", err)
	}
	dbTypeNames := make([]string, len(colTypes))
	for i, ct := range colTypes {
		dbTypeNames[i] = ct.DatabaseTypeName()
	}

	result := &Result{
		Columns: cols,
		Rows:    make([]map[string]any, 0),
	}

	dest := make([]any, len(cols))
	destPtrs := make([]any, len(cols))
	for i := range dest {
		destPtrs[i] = &dest[i]
	}

	for rows.Next() {
		if err := rows.Scan(destPtrs...); err != nil {
			return nil, fmt.Errorf("query: scan row %d: %w", result.RowCount+1, err)
		}
		rowMap := make(map[string]any, len(cols))
		for i, name := range cols {
			rowMap[name] = normalize(dest[i], dbTypeNames[i])
		}
		result.Rows = append(result.Rows, rowMap)
		result.RowCount++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query: iterate rows: %w", err)
	}

	return result, nil
}

func normalize(v any, dbType string) any {
	if v == nil {
		return nil
	}

	switch dbType {
	case "VARIANT", "OBJECT", "ARRAY":
		if s, ok := v.(string); ok {
			var parsed any
			if err := json.Unmarshal([]byte(s), &parsed); err == nil {
				return parsed
			}
			// Fall through and return the raw string if it wasn't valid JSON.
		}
	}

	switch t := v.(type) {
	case time.Time:
		return t.Format(time.RFC3339Nano)
	case []byte:
		return base64.StdEncoding.EncodeToString(t)
	default:
		return t
	}
}
