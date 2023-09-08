package adapters_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/search/adapters"
	"github.com/konstantinfoerster/card-service-go/internal/search/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var repo domain.Repository

func TestIntegrationRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	runner := commontest.NewRunner()
	runner.Run(t, func(t *testing.T) {
		repo = newRepository(t, runner.Connection())

		t.Run("find by name default page", simpleSearch)
		t.Run("find by name last page", simpleSearchLastPage)
		t.Run("find by empty or unknown term", simpleSearchNoResult)
		t.Run("find by card with no image", simpleSearchNoImageURL)
	})
}

func simpleSearch(t *testing.T) {
	result, err := repo.FindByName("ummy Card", domain.NewPage(1, 3))

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.True(t, result.HasMore)
	assert.Len(t, result.Result, 3)
	assert.Equal(t, "http://localhost/images/dummyCard1.png", result.Result[0].Image)
	assert.Equal(t, "http://localhost/images/dummyCard2.png", result.Result[1].Image)
	assert.Equal(t, "http://localhost/images/dummyCard3.png", result.Result[2].Image)
}

func simpleSearchLastPage(t *testing.T) {
	result, err := repo.FindByName("Dummy Card", domain.NewPage(2, 3))

	assert.NoError(t, err)
	assert.Equal(t, 2, result.Page)
	assert.False(t, result.HasMore)
	assert.Len(t, result.Result, 1)
	assert.Equal(t, "http://localhost/images/dummyCard4.png", result.Result[0].Image)
}

func simpleSearchNoImageURL(t *testing.T) {
	result, err := repo.FindByName("No Image Card", domain.NewPage(1, 5))

	assert.NoError(t, err)
	assert.Len(t, result.Result, 2)
	assert.Equal(t, "", result.Result[0].Image)
}

func simpleSearchNoResult(t *testing.T) {
	cases := []struct {
		name       string
		searchTerm string
	}{
		{
			name:       "Empty term",
			searchTerm: "",
		},
		{
			name:       "Space only term",
			searchTerm: " ",
		},
		{
			name:       "unknown term",
			searchTerm: "DoesNotExists",
		},
		{
			name:       "different language",
			searchTerm: "French",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := repo.FindByName(tc.searchTerm, domain.NewPage(1, 10))

			assert.NoError(t, err)
			assert.Equal(t, 1, result.Page)
			assert.False(t, result.HasMore)
			assert.Len(t, result.Result, 0)
		})
	}
}

func newRepository(t *testing.T, con *postgres.DBConnection) domain.Repository {
	t.Helper()

	require.NotNil(t, con)

	cfg := config.Images{
		Host: "http://localhost",
	}

	return adapters.NewRepository(con, cfg)
}
