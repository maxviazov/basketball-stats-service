package repository

import (
	"context"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
)

// pgxLogger адаптирует zerolog.Logger для использования с pgx tracelog.
type pgxLogger struct {
	logger zerolog.Logger
}

// newPgxLogger создает новый адаптер.
// Он также создает дочерний логгер с полем "component":"pgx" для контекста.
func newPgxLogger(logger zerolog.Logger) *pgxLogger {
	l := logger.With().Str("component", "pgx").Logger()
	return &pgxLogger{logger: l}
}

// Log реализует интерфейс tracelog.Logger.
// Он преобразует уровень логирования pgx в уровень zerolog и записывает сообщение.
func (l *pgxLogger) Log(_ context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	if level == tracelog.LogLevelNone {
		return
	}

	var event *zerolog.Event

	switch level {
	case tracelog.LogLevelTrace:
		// Трассируем SQL-запросы на уровне Trace.
		event = l.logger.Trace()
		// Специальная обработка для SQL-запросов для лучшей читаемости.
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
		// Для неизвестных уровней используем Info и добавляем оригинальный уровень в поле.
		event = l.logger.Info().Str("pgx_log_level", level.String())
	}

	// Добавляем остальные поля, если они есть.
	if len(data) > 0 {
		event = event.Fields(data)
	}
	event.Msg(msg)
}
