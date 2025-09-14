package config

import (
	"fmt"

	"github.com/maxviazov/basketball-stats-service/internal/logger"
)

type PostgresConfig struct {
	Host              string `mapstructure:"host"`
	Port              int    `mapstructure:"port"`
	User              string `mapstructure:"user"`
	Password          string `mapstructure:"password"`
	DBName            string `mapstructure:"dbname"`
	SSLMode           string `mapstructure:"sslmode"`
	MaxConns          int32  `mapstructure:"max_conns"`
	MinConns          int32  `mapstructure:"min_conns"`
	MaxConnLifetime   int    `mapstructure:"max_conn_lifetime"`   // seconds
	MaxConnIdleTime   int    `mapstructure:"max_conn_idle_time"`  // seconds
	HealthCheckPeriod int    `mapstructure:"health_check_period"` // seconds
}

// String implements fmt.Stringer with password redaction.
func (p PostgresConfig) String() string {
	masked := ""
	if p.Password != "" {
		masked = "******"
	}
	return fmt.Sprintf("{Host:%s Port:%d User:%s Password:%s DBName:%s SSLMode:%s MaxConns:%d MinConns:%d MaxConnLifetime:%d MaxConnIdleTime:%d HealthCheckPeriod:%d}",
		p.Host, p.Port, p.User, masked, p.DBName, p.SSLMode, p.MaxConns, p.MinConns, p.MaxConnLifetime, p.MaxConnIdleTime, p.HealthCheckPeriod,
	)
}

type Config struct {
	Logger   logger.LoggerConfig `mapstructure:"logger"`
	Postgres PostgresConfig      `mapstructure:"postgres"`
}
