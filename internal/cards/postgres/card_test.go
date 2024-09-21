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

func TestFind(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	cardRepo := postgres.NewCardRepository(connection, cfg)

	cases := []struct {
		name       string
		searchTerm string
		page       cards.Page
		expected   []cards.Card
	}{
		{
			name:       "Empty term",
			searchTerm: "",
			page:       cards.NewPage(1, 2),
			expected: []cards.Card{
				{
					Name:  "Aa Card",
					Set:   cards.Set{Code: "M16", Name: "Magic 2016"},
					Image: cards.Image{URL: "http://localhost/images/dummyCard17.png"},
					ID:    17,
				},
				{
					Name:  "Baa Card",
					Set:   cards.Set{Code: "M16", Name: "Magic 2016"},
					Image: cards.Image{URL: "http://localhost/images/dummyCard16.png"},
					ID:    16,
				},
			},
		},
		{
			name:       "Space only term",
			searchTerm: " ",
			page:       cards.NewPage(1, 2),
			expected: []cards.Card{
				{
					Name:  "Aa Card",
					Set:   cards.Set{Code: "M16", Name: "Magic 2016"},
					Image: cards.Image{URL: "http://localhost/images/dummyCard17.png"},
					ID:    17,
				},
				{
					Name:  "Baa Card",
					Set:   cards.Set{Code: "M16", Name: "Magic 2016"},
					Image: cards.Image{URL: "http://localhost/images/dummyCard16.png"},
					ID:    16,
				},
			},
		},
		{
			name:       "match name on first page",
			searchTerm: "ummy card",
			page:       cards.NewPage(1, 3),
			expected: []cards.Card{
				{
					Name:  "Dummy Card 1",
					Set:   cards.Set{Code: "M10", Name: "Magic 2010"},
					Image: cards.Image{URL: "http://localhost/images/dummyCard1.png"},
					ID:    1,
				},
				{
					Name:  "Dummy Card 2",
					Set:   cards.Set{Code: "M10", Name: "Magic 2010"},
					Image: cards.Image{URL: "http://localhost/images/dummyCard2.png"},
					ID:    2,
				},
				{
					Name:  "Dummy Card 3",
					Set:   cards.Set{Code: "M10", Name: "Magic 2010"},
					Image: cards.Image{URL: "http://localhost/images/dummyCard3.png"},
					ID:    3,
				},
			},
		},
		{
			name:       "match name on last page",
			searchTerm: "ummy card",
			page:       cards.NewPage(2, 3),
			expected: []cards.Card{
				{
					Name:  "Dummy Card 4",
					Set:   cards.Set{Code: "M10", Name: "Magic 2010"},
					Image: cards.Image{URL: "http://localhost/images/dummyCard4.png"},
					ID:    4,
				},
			},
		},
		{
			name:       "no result when image language does not match",
			searchTerm: "French",
			page:       cards.NewPage(1, 2),
			expected:   []cards.Card{},
		},
		{
			name:       "name does not match",
			searchTerm: "DoesNotExists",
			page:       cards.NewPage(1, 2),
			expected:   []cards.Card{},
		},
		{
			name:       "card name match but face name does not match",
			searchTerm: "Double Face",
			page:       cards.NewPage(1, 2),
			expected:   []cards.Card{},
		},
		{
			name:       "match only front face name",
			searchTerm: "front face",
			page:       cards.NewPage(1, 10),
			expected: []cards.Card{
				{
					Name:  "Front Face doubleFace",
					Set:   cards.Set{Code: "M13", Name: "Magic 2013"},
					Image: cards.Image{URL: "http://localhost/images/FrontFace.png"},
					ID:    8,
				},
			},
		},
		{
			name:       "match only back face name",
			searchTerm: "back face",
			page:       cards.NewPage(1, 10),
			expected: []cards.Card{
				{
					Name:  "Back Face doubleFace",
					Set:   cards.Set{Code: "M13", Name: "Magic 2013"},
					Image: cards.Image{URL: "http://localhost/images/BackFace.png"},
					ID:    8,
				},
			},
		},
		{
			name:       "match both faces returns single card",
			searchTerm: "doubleface",
			page:       cards.NewPage(1, 10),
			expected: []cards.Card{
				{
					Name:  "Front Face doubleFace",
					Set:   cards.Set{Code: "M13", Name: "Magic 2013"},
					Image: cards.Image{URL: "http://localhost/images/FrontFace.png"},
					ID:    8,
				},
			},
		},
		{
			name:       "match card without image",
			searchTerm: "No Image Card 1",
			page:       cards.NewPage(1, 10),
			expected: []cards.Card{
				{
					Name:  "No Image Card 1",
					Set:   cards.Set{Code: "M11", Name: "Magic 2011"},
					Image: cards.Image{URL: ""},
					ID:    5,
				},
			},
		},
		{
			name:       "match card without face image",
			searchTerm: "No Image Card 2",
			page:       cards.NewPage(1, 10),
			expected: []cards.Card{
				{
					Name:  "No Image Card 2",
					Set:   cards.Set{Code: "M11", Name: "Magic 2011"},
					Image: cards.Image{URL: "http://localhost/images/noFace.png"},
					ID:    6,
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			filter := cards.NewFilter().WithName(tc.searchTerm)
			result, err := cardRepo.Find(ctx, filter, tc.page)

			require.NoError(t, err)
			assert.Equal(t, tc.page.Page(), result.Page)
			assert.ElementsMatch(t, tc.expected, result.Result)
		})
	}
}

func TestFindWithCollector(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	cardRepo := postgres.NewCardRepository(connection, cfg)

	cases := []struct {
		name       string
		searchTerm string
		expected   []cards.Card
	}{
		{
			name:       "match name with amount",
			searchTerm: "ummy Card",
			expected: []cards.Card{
				{
					Name:   "Dummy Card 1",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard1.png"},
					ID:     1,
					Amount: 3,
				},
				{
					Name:   "Dummy Card 2",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard2.png"},
					ID:     2,
					Amount: 1,
				},
				{
					Name:   "Dummy Card 3",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard3.png"},
					ID:     3,
					Amount: 0,
				},
			},
		},
		{
			name:       "card name match but face name does not match",
			searchTerm: "Double Face",
			expected:   []cards.Card{},
		},
		{
			name:       "match only front face name",
			searchTerm: "Front face ",
			expected: []cards.Card{
				{
					Name:   "Front Face doubleFace",
					Set:    cards.Set{Code: "M13", Name: "Magic 2013"},
					Image:  cards.Image{URL: "http://localhost/images/FrontFace.png"},
					ID:     8,
					Amount: 2,
				},
			},
		},
		{
			name:       "match only back face name",
			searchTerm: "Back Face",
			expected: []cards.Card{
				{
					Name:   "Back Face doubleFace",
					Set:    cards.Set{Code: "M13", Name: "Magic 2013"},
					Image:  cards.Image{URL: "http://localhost/images/BackFace.png"},
					ID:     8,
					Amount: 2,
				},
			},
		},
		{
			name:       "match both faces returns single card",
			searchTerm: "doubleFace",
			expected: []cards.Card{
				{
					Name:   "Front Face doubleFace",
					Set:    cards.Set{Code: "M13", Name: "Magic 2013"},
					Image:  cards.Image{URL: "http://localhost/images/FrontFace.png"},
					ID:     8,
					Amount: 2,
				},
			},
		},
		{
			name:       "match card without image",
			searchTerm: "No Image Card 1",
			expected: []cards.Card{
				{
					Name:   "No Image Card 1",
					Set:    cards.Set{Code: "M11", Name: "Magic 2011"},
					Image:  cards.Image{URL: ""},
					ID:     5,
					Amount: 5,
				},
			},
		},
		{
			name:       "match card without face image",
			searchTerm: "No Image Card 2",
			expected: []cards.Card{
				{
					Name:   "No Image Card 2",
					Set:    cards.Set{Code: "M11", Name: "Magic 2011"},
					Image:  cards.Image{URL: "http://localhost/images/noFace.png"},
					ID:     6,
					Amount: 1,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			f := cards.NewFilter().WithName(tc.searchTerm).WithCollector(collector)
			result, err := cardRepo.Find(ctx, f, cards.NewPage(1, 3))

			require.NoError(t, err)
			assert.ElementsMatch(t, tc.expected, result.Result)
		})
	}
}

func TestFindOnlyCollected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	cardRepo := postgres.NewCardRepository(connection, cfg)
	expected := []cards.Card{
		{
			Name:   "Dummy Card 1",
			Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
			Image:  cards.Image{URL: "http://localhost/images/dummyCard1.png"},
			ID:     1,
			Amount: 3,
		},
		{
			Name:   "Dummy Card 2",
			Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
			Image:  cards.Image{URL: "http://localhost/images/dummyCard2.png"},
			ID:     2,
			Amount: 1,
		},
	}

	ctx := context.Background()
	filter := cards.NewFilter().
		WithName("ummy Card").
		WithCollector(collector).
		WithOnlyCollected()
	result, err := cardRepo.Find(ctx, filter, cards.NewPage(1, 10))

	require.NoError(t, err)
	assert.Len(t, result.Result, 2)
	assert.ElementsMatch(t, expected, result.Result)
}

func TestFindByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	cardRepo := postgres.NewCardRepository(connection, cfg)

	ctx := context.Background()
	exist, err := cardRepo.Exist(ctx, 1)

	require.NoError(t, err)
	assert.True(t, exist)
}

func TestFindByNoneExistingID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	cardRepo := postgres.NewCardRepository(connection, cfg)

	ctx := context.Background()
	exist, err := cardRepo.Exist(ctx, 1000)

	require.NoError(t, err)
	assert.False(t, exist)
}

func TestAddCards(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	cardRepo := postgres.NewCardRepository(connection, cfg)
	item, err := cards.NewCollectable(9, 2)
	require.NoError(t, err)

	ctx := context.Background()
	err = cardRepo.Collect(ctx, item, collector)
	require.NoError(t, err)

	filter := cards.NewFilter().
		WithName("Uncollected Card 1").
		WithCollector(collector).
		WithOnlyCollected()
	page, err := cardRepo.Find(ctx, filter, cards.NewPage(1, 10))
	require.NoError(t, err)

	require.Len(t, page.Result, 1)
	assert.Equal(t, 9, page.Result[0].ID)
	assert.Equal(t, 2, page.Result[0].Amount)
}

func TestAddNoneExistingCardNoError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	cardRepo := postgres.NewCardRepository(connection, cfg)
	noneExistingItem, _ := cards.NewCollectable(1000, 1)

	ctx := context.Background()
	err := cardRepo.Collect(ctx, noneExistingItem, collector)

	require.NoError(t, err)
}

func TestRemoveCards(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	cardRepo := postgres.NewCardRepository(connection, cfg)
	item, err := cards.RemoveItem(10)
	require.NoError(t, err)

	ctx := context.Background()
	err = cardRepo.Remove(ctx, item, collector)
	require.NoError(t, err)

	filter := cards.NewFilter().
		WithName("Remove Collected Card 1").
		WithCollector(collector).
		WithOnlyCollected()
	page, err := cardRepo.Find(ctx, filter, cards.NewPage(1, 10))
	require.NoError(t, err)

	require.Empty(t, page.Result)
}

func TestRemoveUncollectedCardNoError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	cardRepo := postgres.NewCardRepository(connection, cfg)
	noneExistingItem, _ := cards.RemoveItem(2000)

	ctx := context.Background()
	err := cardRepo.Remove(ctx, noneExistingItem, collector)

	require.NoError(t, err)
}
