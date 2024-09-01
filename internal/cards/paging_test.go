package cards_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/stretchr/testify/assert"
)

func TestNewPage(t *testing.T) {
	cases := []struct {
		name       string
		page       cards.Page
		expectPage cards.Page
	}{
		{
			name:       "page 0 fallbacks to default",
			page:       cards.NewPage(0, 10),
			expectPage: cards.NewPage(1, 10),
		},
		{
			name:       "page size 0 fallbacks to default",
			page:       cards.NewPage(1, 0),
			expectPage: cards.NewPage(1, 10),
		},
		{
			name:       "page size fallbacks to limit if exceed",
			page:       cards.NewPage(1, 1000),
			expectPage: cards.NewPage(1, 100),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectPage, tc.page)
		})
	}
}
