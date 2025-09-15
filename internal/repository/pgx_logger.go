package repository

import (
	"context"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
)

// pgxLogger adapts zerolog.Logger to pgx's tracelog interface.
// I keep this tiny and allocation-friendly, only translating levels and passing fields through.
type pgxLogger struct {
	logger zerolog.Logger
}

// newPgxLogger builds a child logger scoped to the pgx component.
// I like to tag component explicitly so SQL noise stays filterable.
func newPgxLogger(logger zerolog.Logger) *pgxLogger {
	l := logger.With().Str("component", "pgx").Logger()
	return &pgxLogger{logger: l}
}

// Log implements tracelog.Logger by mapping pgx levels to zerolog and
// adding commonly useful fields, such as SQL and args, when present.
func (l *pgxLogger) Log(_ context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	if level == tracelog.LogLevelNone {
		return
	}

	var event *zerolog.Event

	switch level {
	case tracelog.LogLevelTrace:
		// I trace SQL at the most verbose level; it’s invaluable during debugging.
		event = l.logger.Trace()
		if sqlVal, ok := data["sql"]; ok {
			if s, ok := sqlVal.(string); ok {
				event = event.Str("sql", s)
			} else {
				event = event.Interface("sql", sqlVal)
			}
			delete(data, "sql")
		}
		if args, ok := data["args"]; ok {
			event = event.Interface("args", args)
			delete(data, "args")
		}
	case tracelog.LogLevelDebug:
		event = l.logger.Debug()
	case tracelog.LogLevelInfo:
		event = l.logger.Info()
	case tracelog.LogLevelWarn:
		event = l.logger.Warn()
	case tracelog.LogLevelError:
		event = l.logger.Error()
	default:
		// If pgx adds new levels, I'll fall back to info and keep the original level as a field.
		event = l.logger.Info().Str("pgx_log_level", level.String())
	}

	if len(data) > 0 {
		event = event.Fields(data)
	}
	event.Msg(msg)
}
