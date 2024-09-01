package postgres_test

import (
	"context"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindByName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	cardRepo := postgres.NewCardRepository(connection, cfg)

	ctx := context.Background()
	result, err := cardRepo.FindByName(ctx, "ummy Card", cards.NewPage(1, 3))

	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Len(t, result.Result, 3)
	assert.True(t, result.HasMore)
	assert.Equal(t, "Dummy Card 1", result.Result[0].Name)
	assert.Equal(t, "http://localhost/images/dummyCard1.png", result.Result[0].Image)
	assert.Equal(t, "Dummy Card 2", result.Result[1].Name)
	assert.Equal(t, "http://localhost/images/dummyCard2.png", result.Result[1].Image)
	assert.Equal(t, "Dummy Card 3", result.Result[2].Name)
	assert.Equal(t, "http://localhost/images/dummyCard3.png", result.Result[2].Image)
}

func TestFindByNameLastPage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	cardRepo := postgres.NewCardRepository(connection, cfg)

	ctx := context.Background()
	result, err := cardRepo.FindByName(ctx, "Dummy Card", cards.NewPage(2, 3))

	require.NoError(t, err)
	assert.False(t, result.HasMore)
	assert.Equal(t, 2, result.Page)
	assert.Len(t, result.Result, 1)
	assert.Equal(t, "http://localhost/images/dummyCard4.png", result.Result[0].Image)
}

func TestFindByNameDoubleFace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	cardRepo := postgres.NewCardRepository(connection, cfg)

	cases := []struct {
		name       string
		searchTerm string
		resultSize int
	}{
		{
			name:       "card name",
			searchTerm: "Double Face",
			resultSize: 0,
		},
		{
			name:       "front face name",
			searchTerm: "Front face ",
			resultSize: 1,
		},
		{
			name:       "back face name",
			searchTerm: "Back Face",
			resultSize: 1,
		},
		{
			name:       "both faces",
			searchTerm: "doubleFace",
			resultSize: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := cardRepo.FindByName(ctx, tc.searchTerm, cards.NewPage(1, 10))

			require.NoError(t, err)
			assert.Len(t, result.Result, tc.resultSize)
		})
	}
}

func TestFindByNameNoImageURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	cardRepo := postgres.NewCardRepository(connection, cfg)

	ctx := context.Background()
	result, err := cardRepo.FindByName(ctx, "No Image Card", cards.NewPage(1, 5))

	require.NoError(t, err)
	assert.Len(t, result.Result, 2)
	assert.Equal(t, "", result.Result[0].Image)
	assert.Equal(t, "http://localhost/images/noFace.png", result.Result[1].Image)
}

func TestFindByNameNoResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	cardRepo := postgres.NewCardRepository(connection, cfg)

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
			ctx := context.Background()
			result, err := cardRepo.FindByName(ctx, tc.searchTerm, cards.NewPage(1, 10))

			require.NoError(t, err)
			assert.Equal(t, 1, result.Page)
			assert.False(t, result.HasMore)
			assert.Empty(t, result.Result)
		})
	}
}

func TestFindByNameAndCollector(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	cardRepo := postgres.NewCardRepository(connection, cfg)

	ctx := context.Background()
	result, err := cardRepo.FindByNameWithAmount(ctx, "ummy Card", collector, cards.NewPage(1, 3))

	require.NoError(t, err)
	assert.Len(t, result.Result, 3)
	assert.Equal(t, "Dummy Card 1", result.Result[0].Name)
	assert.Equal(t, "http://localhost/images/dummyCard1.png", result.Result[0].Image)
	assert.Equal(t, 3, result.Result[0].Amount)
	assert.Equal(t, "Dummy Card 2", result.Result[1].Name)
	assert.Equal(t, "http://localhost/images/dummyCard2.png", result.Result[1].Image)
	assert.Equal(t, 1, result.Result[1].Amount)
	assert.Equal(t, "Dummy Card 3", result.Result[2].Name)
	assert.Equal(t, "http://localhost/images/dummyCard3.png", result.Result[2].Image)
	assert.Empty(t, result.Result[2].Amount)
}

func TestFindByNameAndCollectorDoubleFace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	cardRepo := postgres.NewCardRepository(connection, cfg)

	cases := []struct {
		name       string
		searchTerm string
		resultSize int
		amount     int
	}{
		{
			name:       "card name",
			searchTerm: "Double Face",
			resultSize: 0,
			amount:     2,
		},
		{
			name:       "front face name",
			searchTerm: "Front face ",
			resultSize: 1,
			amount:     2,
		},
		{
			name:       "back face name",
			searchTerm: "Back Face",
			resultSize: 1,
			amount:     2,
		},
		{
			name:       "both faces",
			searchTerm: "doubleFace",
			resultSize: 2,
			amount:     2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := cardRepo.FindByNameWithAmount(ctx, tc.searchTerm, collector, cards.NewPage(1, 10))

			require.NoError(t, err)
			assert.Len(t, result.Result, tc.resultSize)

			for _, r := range result.Result {
				assert.Equal(t, tc.amount, r.Amount)
			}
		})
	}
}

func TestFindByNameAndCollectorNoImageURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	cardRepo := postgres.NewCardRepository(connection, cfg)

	ctx := context.Background()
	result, err := cardRepo.FindByNameWithAmount(ctx, "No Image Card", collector, cards.NewPage(1, 10))

	require.NoError(t, err)
	assert.Len(t, result.Result, 2)
	assert.Equal(t, 5, result.Result[0].Amount)
	assert.Equal(t, "", result.Result[0].Image)
	assert.Equal(t, 1, result.Result[1].Amount)
	assert.Equal(t, "http://localhost/images/noFace.png", result.Result[1].Image)
}
