package config_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig_Defaults(t *testing.T) {
	cfg, err := config.NewConfig("testdata/empty.yaml")

	require.NoError(t, err)
	assert.NotEmpty(t, cfg.Logging.Level)
	assert.NotEmpty(t, cfg.Server.Port)
	assert.NotEmpty(t, cfg.Probes.Port)
	assert.NotEmpty(t, cfg.Auth.HeaderUserID)
	assert.NotEmpty(t, cfg.Auth.HeaderUserEmail)
	assert.NotEmpty(t, cfg.Auth.LoginURL)
	assert.NotEmpty(t, cfg.Auth.LogoutURL)
	assert.False(t, cfg.Auth.TestMode)
}

func TestNewConfig_OverwriteDefaults(t *testing.T) {
	cfg, err := config.NewConfig("testdata/application.yaml")

	require.NoError(t, err)
	assert.Equal(t, "trace", cfg.Logging.Level)
	assert.Equal(t, 13000, cfg.Server.Port)
	assert.Equal(t, 13001, cfg.Probes.Port)
	assert.Equal(t, "X-my-userid", cfg.Auth.HeaderUserID)
	assert.Equal(t, "X-my-useremail", cfg.Auth.HeaderUserEmail)
	assert.Equal(t, "https://localhost/oidc/login", cfg.Auth.LoginURL)
	assert.Equal(t, "https://localhost/oidc/logout", cfg.Auth.LogoutURL)
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
