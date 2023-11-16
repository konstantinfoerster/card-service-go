package domain_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewPage(t *testing.T) {
	cases := []struct {
		name       string
		page       domain.Page
		expectPage domain.Page
	}{
		{
			name:       "Page 0 fallbacks to default",
			page:       domain.NewPage(0, 10),
			expectPage: domain.NewPage(1, 10),
		},
		{
			name:       "Size 0 fallbacks to default",
			page:       domain.NewPage(1, 0),
			expectPage: domain.NewPage(1, 10),
		},
		{
			name:       "To large size fallbacks to limit",
			page:       domain.NewPage(1, 1000),
			expectPage: domain.NewPage(1, 100),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectPage, tc.page)
		})
	}
}

func TestHasMore(t *testing.T) {
	cases := []struct {
		name       string
		page       domain.Page
		resultSize int
		expectMore bool
	}{
		{
			name:       "Has more if result is bigger then current page size",
			page:       domain.NewPage(1, 10),
			resultSize: 11,
			expectMore: true,
		},
		{
			name:       "Possibly more if result and page size is equal",
			page:       domain.NewPage(1, 10),
			resultSize: 10,
			expectMore: true,
		},
		{
			name:       "No more if result is zero",
			page:       domain.NewPage(1, 10),
			resultSize: 0,
			expectMore: false,
		},
		{
			name:       "No more if result is less then page size",
			page:       domain.NewPage(2, 10),
			resultSize: 9,
			expectMore: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectMore, domain.HasMore(tc.page, tc.resultSize))
		})
	}
}
