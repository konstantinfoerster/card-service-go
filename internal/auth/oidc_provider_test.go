package auth_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromConfiguration(t *testing.T) {
	cfg := auth.OidcConfig{
		Provider: map[string]auth.ProviderConfig{
			"google": {
				ClientID: "test-client-id",
				Secret:   "test-secret-0",
			},
		},
	}

	provider, err := auth.FromConfiguration(cfg)

	require.NoError(t, err)
	assert.Len(t, provider, 1)
}

func TestFromConfigurationMisconfigured(t *testing.T) {
	cases := []struct {
		name string
		cfg  auth.OidcConfig
	}{
		{
			name: "no provider configure",
			cfg: auth.OidcConfig{
				Provider: map[string]auth.ProviderConfig{},
			},
		},
		{
			name: "no client id",
			cfg: auth.OidcConfig{
				Provider: map[string]auth.ProviderConfig{
					"google": {
						Secret: "test-secret-0",
					},
				},
			},
		},
		{
			name: "no secret",
			cfg: auth.OidcConfig{
				Provider: map[string]auth.ProviderConfig{
					"google": {
						ClientID: "test-client-id",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := auth.FromConfiguration(tc.cfg)

			require.Error(t, err)
		})
	}
}
