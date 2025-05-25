package auth_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromConfiguration(t *testing.T) {
	cfg := auth.Config{
		Provider: map[string]auth.ProviderCfg{
			"google": {
				ClientID: "test-client-id",
				Secret:   "test-secret-0",
			},
		},
	}

	provider, err := auth.FromConfiguration(cfg)
	require.NoError(t, err)
	p, err := provider.Find("google")

	require.NoError(t, err)
	assert.Equal(t, "google", p.GetName())
}

func TestFromConfigurationMisconfigured(t *testing.T) {
	cases := []struct {
		name string
		cfg  auth.Config
	}{
		{
			name: "no provider configure",
			cfg: auth.Config{
				Provider: map[string]auth.ProviderCfg{},
			},
		},
		{
			name: "no client id",
			cfg: auth.Config{
				Provider: map[string]auth.ProviderCfg{
					"google": {
						Secret: "test-secret-0",
					},
				},
			},
		},
		{
			name: "no secret",
			cfg: auth.Config{
				Provider: map[string]auth.ProviderCfg{
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
