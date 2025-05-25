package web_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/stretchr/testify/assert"
)

func TestAddr(t *testing.T) {
	cases := []struct {
		name     string
		cfg      web.Config
		expected string
	}{
		{
			name: "host with port",
			cfg: web.Config{
				Host: "localhost",
				Port: 3000,
			},
			expected: "localhost:3000",
		},
		{
			name: "host only",
			cfg: web.Config{
				Host: "localhost",
			},
			expected: "localhost:3000",
		},
		{
			name: "port only",
			cfg: web.Config{
				Host: "",
				Port: 3000,
			},
			expected: ":3000",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.cfg.Addr())
		})
	}
}
