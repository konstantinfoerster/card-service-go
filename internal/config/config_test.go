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

	assert.NotEmpty(t, cfg.Auth.Session.Name)
	assert.NotEmpty(t, cfg.Auth.Session.SameSite)
	assert.Greater(t, cfg.Auth.Session.ExpiresIn, time.Second)

	assert.NotEmpty(t, cfg.Auth.State.Name)
	assert.NotEmpty(t, cfg.Auth.State.SameSite)
	assert.Greater(t, cfg.Auth.State.ExpiresIn, time.Second)
}

func TestNewConfig_OverwriteDefaults(t *testing.T) {
	cfg, err := config.NewConfig("testdata/application.yaml")

	require.NoError(t, err)
	assert.Equal(t, "trace", cfg.Logging.Level)

	assert.Equal(t, "SESSION_TEST", cfg.Auth.Session.Name)
	assert.Equal(t, "none", cfg.Auth.Session.SameSite)
	assert.Equal(t, 2*time.Hour, cfg.Auth.Session.ExpiresIn)

	assert.Equal(t, "STATE_TEST", cfg.Auth.State.Name)
	assert.Equal(t, "none", cfg.Auth.State.SameSite)
	assert.Equal(t, 2*time.Minute, cfg.Auth.State.ExpiresIn)
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
