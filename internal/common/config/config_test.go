package config_test

import (
	"testing"
	"time"

	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestNewConfig_Defaults(t *testing.T) {
	cfg, err := config.NewConfig("testdata/empty.yaml")

	require.NoError(t, err)
	assert.NotEmpty(t, cfg.Logging.Level)
	assert.NotEmpty(t, cfg.Oidc.SessionCookieName)
	assert.Greater(t, cfg.Oidc.StateCookieAge, time.Second)
}

func TestNewConfig_OverwriteDefaults(t *testing.T) {
	cfg, err := config.NewConfig("testdata/application.yaml")

	require.NoError(t, err)
	assert.Equal(t, "http://localhost:3000/test", cfg.Oidc.RedirectURI)
	assert.Equal(t, "trace", cfg.Logging.Level)
	assert.Equal(t, "SESSION_TEST", cfg.Oidc.SessionCookieName)
	assert.Equal(t, time.Hour*2, cfg.Oidc.StateCookieAge)
}
