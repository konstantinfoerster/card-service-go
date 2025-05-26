package config_test

import (
	"testing"
	"time"

	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig_Defaults(t *testing.T) {
	cfg, err := config.NewConfig("testdata/empty.yaml")

	require.NoError(t, err)
	assert.NotEmpty(t, cfg.Server.Port)
	assert.NotEmpty(t, cfg.Server.TemplateDir)
	assert.NotEmpty(t, cfg.Probes.Port)
	assert.NotEmpty(t, cfg.Logging.Level)
	assert.NotEmpty(t, cfg.Oidc.SessionCookieName)
	assert.Greater(t, cfg.Oidc.StateCookieAge, time.Second)
}

func TestNewConfig_OverwriteDefaults(t *testing.T) {
	cfg, err := config.NewConfig("testdata/application.yaml")

	require.NoError(t, err)
	assert.Equal(t, "trace", cfg.Logging.Level)
	assert.Equal(t, "SESSION_TEST", cfg.Oidc.SessionCookieName)
	assert.Equal(t, time.Hour*2, cfg.Oidc.StateCookieAge)
}

func TestNewConfig_NotAFile(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{
			name: "directory",
			path: "testdata",
		},
		{
			name: "file not exist",
			path: "testdata/notfound.yaml",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := config.NewConfig(tc.path)

			require.ErrorIs(t, err, config.ErrReadFile)
		})
	}
}
