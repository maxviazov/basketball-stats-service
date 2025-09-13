package test

import (
	"os"
	"testing"

	logpkg "github.com/maxviazov/basketball-stats-service/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name           string
		config         *logpkg.LoggerConfig
		expectError    bool
		validateOutput func(zerolog.Logger) bool
	}{
		{
			name: "valid production environment",
			config: &logpkg.LoggerConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Env:            "prod",
				Level:          "info",
				TimeField:      "timestamp",
				TimeFormat:     zerolog.TimeFormatUnix,
				Fields:         map[string]interface{}{"key": "value"},
				WithCaller:     false,
				Stacktrace:     false,
			},
			expectError: false,
			validateOutput: func(logger zerolog.Logger) bool {
				return zerolog.GlobalLevel() == zerolog.InfoLevel
			},
		},
		{
			name: "invalid configuration - wrong env",
			config: &logpkg.LoggerConfig{
				ServiceName: "bad-service",
				Env:         "wrong-env", // not allowed by validator
				Level:       "debug",
			},
			expectError:    true,
			validateOutput: nil,
		},
		{
			name: "valid development environment with debug level",
			config: &logpkg.LoggerConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Env:            "dev",
				Level:          "debug",
				TimeField:      "timestamp",
				TimeFormat:     zerolog.TimeFormatUnix,
				Fields:         map[string]interface{}{"key": "value"},
				WithCaller:     true,
				Stacktrace:     true,
			},
			expectError: false,
			validateOutput: func(logger zerolog.Logger) bool {
				return true
			},
		},
		{
			name: "invalid log level",
			config: &logpkg.LoggerConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Env:            "prod",
				Level:          "invalid-level", // not allowed
				TimeField:      "timestamp",
				TimeFormat:     zerolog.TimeFormatUnix,
			},
			expectError:    true,
			validateOutput: nil,
		},
		{
			name: "valid staging environment",
			config: &logpkg.LoggerConfig{
				ServiceName:    "test-service",
				ServiceVersion: "2.0.0",
				Env:            "staging",
				Level:          "warn",
				TimeField:      "time",
				TimeFormat:     zerolog.TimeFormatUnix,
				WithCaller:     false,
				Stacktrace:     true,
			},
			expectError: false,
			validateOutput: func(logger zerolog.Logger) bool {
				return zerolog.GlobalLevel() == zerolog.WarnLevel
			},
		},
		{
			name: "valid development environment without debug",
			config: &logpkg.LoggerConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Env:            "dev",
				Level:          "info",
				TimeField:      "time",
				TimeFormat:     zerolog.TimeFormatUnix,
				WithCaller:     false,
				Stacktrace:     false,
			},
			expectError: false,
			validateOutput: func(logger zerolog.Logger) bool {
				return zerolog.GlobalLevel() == zerolog.InfoLevel
			},
		},
		{
			name: "valid production environment with additional fields",
			config: &logpkg.LoggerConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.1",
				Env:            "prod",
				Level:          "error",
				TimeField:      "timestamp",
				TimeFormat:     zerolog.TimeFormatUnix,
				Fields:         map[string]interface{}{"customField": "customValue"},
				WithCaller:     true,
				Stacktrace:     true,
			},
			expectError: false,
			validateOutput: func(logger zerolog.Logger) bool {
				return zerolog.GlobalLevel() == zerolog.ErrorLevel
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			l, err := logpkg.New(test.config)
			if test.expectError {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
				if test.validateOutput != nil {
					assert.True(t, test.validateOutput(l))
				}
			}
		})
	}

	t.Run("debug log file creation", func(t *testing.T) {
		config := &logpkg.LoggerConfig{
			ServiceName:    "integration-test",
			ServiceVersion: "1.0.0",
			Env:            "dev",
			Level:          "debug",
			TimeField:      "timestamp",
			TimeFormat:     zerolog.TimeFormatUnix,
		}

		_, err := logpkg.New(config)
		assert.NoError(t, err)

		_, statErr := os.Stat("logs/debug.log")
		assert.NoError(t, statErr)

		t.Cleanup(func() {
			if err := os.Remove("logs/debug.log"); err != nil && !os.IsNotExist(err) {
				t.Logf("cleanup failed: %v", err)
			}
		})
	})
}
