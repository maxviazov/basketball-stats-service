package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

type LoggerConfig struct {
	Level              string                 `json:"level,omitempty" validate:"oneof=debug info warn error"`
	Format             string                 `json:"format,omitempty" validate:"oneof=json console"`
	OutputTarget       string                 `json:"outputTarget,omitempty" validate:"oneof=stdout stderr"`
	TimeField          string                 `json:"timeField,omitempty"`
	TimeFormat         string                 `json:"timeFormat,omitempty" validate:"oneof=rfc3339 rfc3339nano unix unix_ms"`
	ServiceName        string                 `json:"serviceName,omitempty"`
	ServiceVersion     string                 `json:"serviceVersion,omitempty"`
	Env                string                 `json:"env,omitempty" validate:"oneof=dev staging prod"`
	WithCaller         bool                   `json:"withCaller,omitempty"`
	Stacktrace         bool                   `json:"stacktrace,omitempty"`
	StacktraceMinLevel string                 `json:"stacktraceMinLevel,omitempty" validate:"oneof=debug info warn error fatal panic"`
	Fields             map[string]interface{} `json:"fields,omitempty"`
}

func New(logg *LoggerConfig) (logger zerolog.Logger, err error) {
	logg.setDefaults()

	v := validator.New()
	if err = v.Struct(logg); err != nil {
		return logger, fmt.Errorf("logger config validation error: %w", err)
	}

	// apply time settings from config
	zerolog.TimestampFieldName = logg.TimeField
	zerolog.TimeFieldFormat = logg.TimeFormat

	// choose writer based on environment and level
	switch logg.Env {
	case "prod", "staging":
		// production-like environments: JSON logs only, stdout is king
		writer := os.Stdout
		logger = zerolog.New(writer).
			With().
			Timestamp().
			Str("service", logg.ServiceName).
			Str("version", logg.ServiceVersion).
			Str("env", logg.Env).
			Logger()

	case "dev":
		if logg.Level == "debug" {
			// development + debug: console for humans, file for full history
			consoleWriter := zerolog.ConsoleWriter{
				Out:        os.Stderr,
				TimeFormat: logg.TimeFormat,
			}

			logPath := "logs/debug.log"
			// make sure directory exists; don't crash if it fails
			if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
				logger = zerolog.New(consoleWriter).
					With().
					Timestamp().
					Str("service", logg.ServiceName).
					Str("version", logg.ServiceVersion).
					Str("env", logg.Env).
					Logger()
			} else {
				file, ferr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
				if ferr != nil {
					// fallback to console only if file cannot be opened
					logger = zerolog.New(consoleWriter).
						With().
						Timestamp().
						Str("service", logg.ServiceName).
						Str("version", logg.ServiceVersion).
						Str("env", logg.Env).
						Logger()
				} else {
					writer := zerolog.MultiLevelWriter(consoleWriter, file)
					logger = zerolog.New(writer).
						With().
						Timestamp().
						Str("service", logg.ServiceName).
						Str("version", logg.ServiceVersion).
						Str("env", logg.Env).
						Logger()
				}
			}
		} else {
			// development + info/warn/error: console only
			consoleWriter := zerolog.ConsoleWriter{
				Out:        os.Stderr,
				TimeFormat: logg.TimeFormat,
			}
			logger = zerolog.New(consoleWriter).
				With().
				Timestamp().
				Str("service", logg.ServiceName).
				Str("version", logg.ServiceVersion).
				Str("env", logg.Env).
				Logger()
		}
	}

	// add optional extras in a clean linear flow
	if logg.WithCaller {
		logger = logger.With().Caller().Logger()
	}
	if logg.Stacktrace {
		logger = logger.With().Stack().Logger()
	}
	if len(logg.Fields) > 0 {
		logger = logger.With().Fields(logg.Fields).Logger()
	}

	// set log level globally (important: must be after ParseLevel)
	level, err := zerolog.ParseLevel(logg.Level)
	if err != nil {
		return logger, err
	}
	zerolog.SetGlobalLevel(level)

	return logger, nil
}

func (c *LoggerConfig) setDefaults() {
	// environment default
	if c.Env == "" {
		c.Env = "prod"
	}

	// level defaults depend on environment
	if c.Level == "" {
		if c.Env == "dev" {
			c.Level = "debug"
		} else {
			c.Level = "info"
		}
	}

	// format defaults
	if c.Format == "" {
		if c.Env == "dev" {
			c.Format = "console"
		} else {
			c.Format = "json"
		}
	}

	// output target default
	if c.OutputTarget == "" {
		c.OutputTarget = "stdout"
	}

	// time defaults
	if c.TimeField == "" {
		c.TimeField = "ts"
	}
	if c.TimeFormat == "" {
		c.TimeFormat = "rfc3339nano"
	}

	// caller & stacktrace defaults
	if !c.WithCaller && c.Env == "dev" {
		c.WithCaller = true
	}
	if !c.Stacktrace && c.Env != "dev" {
		c.Stacktrace = true
	}
	if c.StacktraceMinLevel == "" {
		c.StacktraceMinLevel = "error"
	}

	// service defaults
	if c.ServiceName == "" {
		c.ServiceName = "basketball-stats-service"
	}
	if c.ServiceVersion == "" {
		c.ServiceVersion = "0.0.1"
	}

	// ensure fields map is not nil
	if c.Fields == nil {
		c.Fields = make(map[string]interface{})
	}
}
