package domain_test

import (
	"github.com/konstantinfoerster/card-service/internal/search/domain"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHasMore(t *testing.T) {
	cases := []struct {
		name  string
		page  domain.Page
		total int
		want  bool
	}{
		{
			name:  "First page zero, has one more",
			page:  domain.NewPage(0, 10),
			total: 11,
			want:  true,
		},
		{
			name:  "First page, has one more",
			page:  domain.NewPage(1, 10),
			total: 11,
			want:  true,
		},
		{
			name:  "First page, has equal",
			page:  domain.NewPage(1, 10),
			total: 10,
			want:  false,
		},
		{
			name:  "First page, has nothing",
			page:  domain.NewPage(1, 10),
			total: 0,
			want:  false,
		},
		{
			name:  "Second page, has less than requested",
			page:  domain.NewPage(2, 10),
			total: 9,
			want:  false,
		},
		{
			name:  "Second page, has one more",
			page:  domain.NewPage(2, 10),
			total: 21,
			want:  true,
		},
		{
			name:  "Second page, has equal",
			page:  domain.NewPage(2, 10),
			total: 20,
			want:  false,
		},
		{
			name:  "Second page, has nothing",
			page:  domain.NewPage(2, 10),
			total: 0,
			want:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, domain.HasMore(tc.page, tc.total))
		})
	}
}
