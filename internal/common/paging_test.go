package common_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestNewPage(t *testing.T) {
	cases := []struct {
		name       string
		page       common.Page
		expectPage common.Page
	}{
		{
			name:       "page 0 fallbacks to default",
			page:       common.NewPage(0, 10),
			expectPage: common.NewPage(1, 10),
		},
		{
			name:       "page size 0 fallbacks to default",
			page:       common.NewPage(1, 0),
			expectPage: common.NewPage(1, 10),
		},
		{
			name:       "page size fallbacks to limit if exceed",
			page:       common.NewPage(1, 1000),
			expectPage: common.NewPage(1, 100),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectPage, tc.page)
		})
	}
}
