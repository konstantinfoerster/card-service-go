package config_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	"github.com/stretchr/testify/assert"
)

func TestDatabaseConnectionURL(t *testing.T) {
	cfg := config.Database{
		Host:     "localhost",
		Port:     "5432",
		Database: "test",
		Username: "Tester",
		Password: "secret",
	}

	assert.Equal(t, "postgres://Tester:secret@localhost:5432/test", cfg.ConnectionURL())
}

func TestServerAddr(t *testing.T) {
	cfg := config.Server{
		Host: "localhost",
		Port: 3000,
	}

	assert.Equal(t, "localhost:3000", cfg.Addr())
}

func TestServerAddrHostOnly(t *testing.T) {
	cfg := config.Server{
		Host: "localhost",
	}

	assert.Equal(t, "localhost:0", cfg.Addr())
}

func TestServerAddrPortOnly(t *testing.T) {
	cfg := config.Server{
		Port: 3000,
	}

	assert.Equal(t, ":3000", cfg.Addr())
}

func TestLoggingLevel(t *testing.T) {
	cfg := config.Logging{
		Level: "ERROR",
	}

	assert.Equal(t, "error", cfg.LevelOrDefault())
}

func TestLoggingLevelDefaultWhenUnset(t *testing.T) {
	cfg := config.Logging{}

	assert.Equal(t, "info", cfg.LevelOrDefault())
}

func TestLoggingLevelDefaultWhenEmpty(t *testing.T) {
	cfg := config.Logging{
		Level: "  ",
	}

	assert.Equal(t, "info", cfg.LevelOrDefault())
}
