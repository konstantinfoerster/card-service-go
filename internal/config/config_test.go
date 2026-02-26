package config_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfigs_EmptyFile(t *testing.T) {
	cfg, err := config.ReadConfigs("testdata/empty.yaml")

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

func TestReadConfigs_OverwriteDefaults(t *testing.T) {
	cfg, err := config.ReadConfigs("testdata/application.yaml")

	require.NoError(t, err)
	assert.Equal(t, "trace", cfg.Logging.Level)
	assert.Equal(t, 13000, cfg.Server.Port)
	assert.Equal(t, 13001, cfg.Probes.Port)
	assert.True(t, cfg.Auth.TestMode)
	assert.Equal(t, "X-my-userid", cfg.Auth.HeaderUserID)
	assert.Equal(t, "X-my-useremail", cfg.Auth.HeaderUserEmail)
	assert.Equal(t, "https://localhost/oidc/login", cfg.Auth.LoginURL)
	assert.Equal(t, "https://localhost/oidc/logout", cfg.Auth.LogoutURL)
}

func TestReadConfigs_MultipleFilesOverwrite(t *testing.T) {
	cfg, err := config.ReadConfigs(
		"testdata/application.yaml",
		"testdata/application-dev.yaml",
	)

	require.NoError(t, err)
	assert.Equal(t, 8443, cfg.Server.Port)
	assert.True(t, cfg.Server.TLS.Enabled)
	assert.Equal(t, "app.crt", cfg.Server.TLS.CertFile)
	assert.Equal(t, "tester", cfg.Database.Username)
	assert.Equal(t, "s3cr3t", cfg.Database.Password)
	assert.Equal(t, "https://localhost:8443/dev/oidc/login", cfg.Auth.LoginURL)
	assert.Equal(t, "https://localhost:8443/dev/oidc/logout", cfg.Auth.LogoutURL)
}

func TestReadConfigs_NotAFile(t *testing.T) {
	cases := []struct {
		name string
		path []string
	}{
		{
			name: "directory",
			path: []string{"testdata"},
		},
		{
			name: "file not exist",
			path: []string{"testdata/notfound.yaml"},
		},
		{
			name: "second file not exist",
			path: []string{
				"testdata/application.yaml",
				"testdata/notfound.yaml",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := config.ReadConfigs(tc.path...)

			require.ErrorIs(t, err, config.ErrReadFile)
		})
	}
}
