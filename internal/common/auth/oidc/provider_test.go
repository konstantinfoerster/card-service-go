package oidc_test

import (
	"net/http"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromConfiguration(t *testing.T) {
	cfg := config.Oidc{
		Provider: map[string]config.Provider{
			"google": {
				ClientID: "test-client-id",
				Secret:   "test-secret-0",
			},
		},
	}

	provider, err := oidc.FromConfiguration(cfg, &http.Client{})

	require.NoError(t, err)
	assert.Len(t, provider, 1)
}

func TestFromConfigurationMisconfigured(t *testing.T) {
	cases := []struct {
		name   string
		cfg    config.Oidc
		client *http.Client
	}{
		{
			name: "no client",
			cfg: config.Oidc{
				Provider: map[string]config.Provider{
					"google": {
						ClientID: "test-client-id",
						Secret:   "test-secret-0",
					},
				},
			},
			client: nil,
		},
		{
			name: "no provider configure",
			cfg: config.Oidc{
				Provider: map[string]config.Provider{},
			},
			client: &http.Client{},
		},
		{
			name: "no client id",
			cfg: config.Oidc{
				Provider: map[string]config.Provider{
					"google": {
						Secret: "test-secret-0",
					},
				},
			},
			client: &http.Client{},
		},
		{
			name: "no secret",
			cfg: config.Oidc{
				Provider: map[string]config.Provider{
					"google": {
						ClientID: "test-client-id",
					},
				},
			},
			client: &http.Client{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := oidc.FromConfiguration(tc.cfg, tc.client)

			require.Error(t, err)
		})
	}
}
