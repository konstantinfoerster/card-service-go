package domain_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/search/domain"
	"github.com/stretchr/testify/assert"
)

func TestHasMore(t *testing.T) {
	cases := []struct {
		name    string
		page    domain.Page
		total   int
		hasMore bool
	}{
		{
			name:    "First page zero, has one more",
			page:    domain.NewPage(0, 10),
			total:   11,
			hasMore: true,
		},
		{
			name:    "First page, has one more",
			page:    domain.NewPage(1, 10),
			total:   11,
			hasMore: true,
		},
		{
			name:    "First page, has equal",
			page:    domain.NewPage(1, 10),
			total:   10,
			hasMore: false,
		},
		{
			name:    "First page, has nothing",
			page:    domain.NewPage(1, 10),
			total:   0,
			hasMore: false,
		},
		{
			name:    "Second page, has less than requested",
			page:    domain.NewPage(2, 10),
			total:   9,
			hasMore: false,
		},
		{
			name:    "Second page, has one more",
			page:    domain.NewPage(2, 10),
			total:   21,
			hasMore: true,
		},
		{
			name:    "Second page, has equal",
			page:    domain.NewPage(2, 10),
			total:   20,
			hasMore: false,
		},
		{
			name:    "Second page, has nothing",
			page:    domain.NewPage(2, 10),
			total:   0,
			hasMore: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.hasMore, domain.HasMore(tc.page, tc.total))
		})
	}
}
