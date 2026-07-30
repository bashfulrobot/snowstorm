package cli

import (
	"context"
	"database/sql"

	"github.com/bashfulrobot/snowstorm/internal/snow"
)

// connect opens a Snowflake connection using the command's persistent flags.
func connect(ctx context.Context) (*sql.DB, error) {
	return snow.Connect(ctx, snow.Options{
		ConnectionName: flagConnection,
		Home:           flagHome,
		PingTimeout:    flagTimeout,
	})
}
