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
	assert.NotEmpty(t, cfg.Server.Port)
	assert.NotEmpty(t, cfg.Server.TemplateDir)
	assert.NotEmpty(t, cfg.Server.Auth.HeaderUserID)
	assert.NotEmpty(t, cfg.Server.Auth.HeaderUserEmail)
	assert.NotEmpty(t, cfg.Server.Auth.LoginURL)
	assert.NotEmpty(t, cfg.Server.Auth.LogoutURL)
	assert.False(t, cfg.Server.Auth.TestMode)
	assert.NotEmpty(t, cfg.Probes.Port)
	assert.NotEmpty(t, cfg.Logging.Level)
}

func TestNewConfig_OverwriteDefaults(t *testing.T) {
	cfg, err := config.NewConfig("testdata/application.yaml")

	require.NoError(t, err)
	assert.Equal(t, "trace", cfg.Logging.Level)

	assert.Equal(t, "X-my-userid", cfg.Server.Auth.HeaderUserID)
	assert.Equal(t, "X-my-useremail", cfg.Server.Auth.HeaderUserEmail)
	assert.Equal(t, "https://localhost/oidc/login", cfg.Server.Auth.LoginURL)
	assert.Equal(t, "https://localhost/oidc/logout", cfg.Server.Auth.LogoutURL)
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
