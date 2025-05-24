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
					Name:   "Aa Card",
					Set:    cards.Set{Code: "M16", Name: "Magic 2016"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard17.png"},
					ID:     cards.NewID(17).WithFace(18),
					Number: "30",
				},
				{
					Name:   "Baa Card",
					Set:    cards.Set{Code: "M16", Name: "Magic 2016"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard16.png"},
					ID:     cards.NewID(16).WithFace(17),
					Number: "20",
				},
			},
		},
		{
			name:       "Space only term",
			searchTerm: " ",
			page:       cards.NewPage(1, 2),
			expected: []cards.Card{
				{
					Name:   "Aa Card",
					Set:    cards.Set{Code: "M16", Name: "Magic 2016"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard17.png"},
					ID:     cards.NewID(17).WithFace(18),
					Number: "30",
				},
				{
					Name:   "Baa Card",
					Set:    cards.Set{Code: "M16", Name: "Magic 2016"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard16.png"},
					ID:     cards.NewID(16).WithFace(17),
					Number: "20",
				},
			},
		},
		{
			name:       "match name on first page",
			searchTerm: "ummy card",
			page:       cards.NewPage(1, 3),
			expected: []cards.Card{
				{
					Name:   "Dummy Card 1",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard1.png"},
					ID:     cards.NewID(1).WithFace(1),
					Number: "1",
				},
				{
					Name:   "Dummy Card 2",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard2.png"},
					ID:     cards.NewID(2).WithFace(2),
					Number: "2",
				},
				{
					Name:   "Dummy Card 3",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard3.png"},
					ID:     cards.NewID(3).WithFace(3),
					Number: "3",
				},
			},
		},
		{
			name:       "match name on last page",
			searchTerm: "ummy card",
			page:       cards.NewPage(2, 3),
			expected: []cards.Card{
				{
					Name:   "Dummy Card 4",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard4.png"},
					ID:     cards.NewID(4).WithFace(4),
					Number: "4",
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
					Name:   "Front Face doubleFace",
					Set:    cards.Set{Code: "M13", Name: "Magic 2013"},
					Image:  cards.Image{URL: "http://localhost/images/FrontFace.png"},
					ID:     cards.NewID(8).WithFace(8),
					Number: "1",
				},
			},
		},
		{
			name:       "match only back face name",
			searchTerm: "back face",
			page:       cards.NewPage(1, 10),
			expected: []cards.Card{
				{
					Name:   "Back Face doubleFace",
					Set:    cards.Set{Code: "M13", Name: "Magic 2013"},
					Image:  cards.Image{URL: "http://localhost/images/BackFace.png"},
					ID:     cards.NewID(8).WithFace(9),
					Number: "1",
				},
			},
		},
		{
			name:       "match both faces returns single card",
			searchTerm: "doubleface",
			page:       cards.NewPage(1, 10),
			expected: []cards.Card{
				{
					Name:   "Front Face doubleFace",
					Set:    cards.Set{Code: "M13", Name: "Magic 2013"},
					Image:  cards.Image{URL: "http://localhost/images/FrontFace.png"},
					ID:     cards.NewID(8).WithFace(8),
					Number: "1",
				},
			},
		},
		{
			name:       "match card without image",
			searchTerm: "No Image Card 1",
			page:       cards.NewPage(1, 10),
			expected: []cards.Card{
				{
					Name:   "No Image Card 1",
					Set:    cards.Set{Code: "M11", Name: "Magic 2011"},
					Image:  cards.Image{URL: ""},
					ID:     cards.NewID(5).WithFace(5),
					Number: "1",
				},
			},
		},
		{
			name:       "match card without image face ID",
			searchTerm: "No Image Card 2",
			page:       cards.NewPage(1, 10),
			expected: []cards.Card{
				{
					Name:   "No Image Card 2",
					Set:    cards.Set{Code: "M11", Name: "Magic 2011"},
					ID:     cards.NewID(6).WithFace(6),
					Number: "2",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			filter := cards.NewFilter().
				WithName(tc.searchTerm).
				WithLanguage(cards.DefaultLang)
			result, err := cardRepo.Find(ctx, filter, tc.page)

			require.NoError(t, err)
			require.Len(t, result.Result, len(tc.expected))
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
		name      string
		filter    cards.Filter
		collector cards.Collector
		expected  []cards.Card
	}{
		{
			name: "match name with amount",
			filter: cards.NewFilter().
				WithName("ummy Card").
				WithCollector(collector),
			expected: []cards.Card{
				{
					Name:   "Dummy Card 1",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard1.png"},
					ID:     cards.NewID(1).WithFace(1),
					Amount: 3,
					Number: "1",
				},
				{
					Name:   "Dummy Card 2",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard2.png"},
					ID:     cards.NewID(2).WithFace(2),
					Amount: 1,
					Number: "2",
				},
				{
					Name:   "Dummy Card 3",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard3.png"},
					ID:     cards.NewID(3).WithFace(3),
					Amount: 0,
					Number: "3",
				},
			},
		},
		{
			name: "card name match but face name does not match",
			filter: cards.NewFilter().
				WithName("Double Face").
				WithCollector(collector),
			expected: []cards.Card{},
		},
		{
			name: "match only front face name",
			filter: cards.NewFilter().
				WithName("Front Face").
				WithCollector(collector),
			expected: []cards.Card{
				{
					Name:   "Front Face doubleFace",
					Set:    cards.Set{Code: "M13", Name: "Magic 2013"},
					Image:  cards.Image{URL: "http://localhost/images/FrontFace.png"},
					ID:     cards.NewID(8).WithFace(8),
					Amount: 2,
					Number: "1",
				},
			},
		},
		{
			name: "match only back face name",
			filter: cards.NewFilter().
				WithName("Back Face").
				WithCollector(collector),
			expected: []cards.Card{
				{
					Name:   "Back Face doubleFace",
					Set:    cards.Set{Code: "M13", Name: "Magic 2013"},
					Image:  cards.Image{URL: "http://localhost/images/BackFace.png"},
					ID:     cards.NewID(8).WithFace(9),
					Amount: 2,
					Number: "1",
				},
			},
		},
		{
			name: "match both faces returns single card",
			filter: cards.NewFilter().
				WithName("doubleface").
				WithCollector(collector),
			expected: []cards.Card{
				{
					Name:   "Front Face doubleFace",
					Set:    cards.Set{Code: "M13", Name: "Magic 2013"},
					Image:  cards.Image{URL: "http://localhost/images/FrontFace.png"},
					ID:     cards.NewID(8).WithFace(8),
					Amount: 2,
					Number: "1",
				},
			},
		},
		{
			name: "match card without image",
			filter: cards.NewFilter().
				WithName("No Image Card 1").
				WithCollector(collector),
			expected: []cards.Card{
				{
					Name:   "No Image Card 1",
					Set:    cards.Set{Code: "M11", Name: "Magic 2011"},
					Image:  cards.Image{URL: ""},
					ID:     cards.NewID(5).WithFace(5),
					Amount: 5,
					Number: "1",
				},
			},
		},
		{
			name: "match card without image face ID",
			filter: cards.NewFilter().
				WithName("No Image Card 2").
				WithCollector(collector),
			expected: []cards.Card{
				{
					Name:   "No Image Card 2",
					Set:    cards.Set{Code: "M11", Name: "Magic 2011"},
					ID:     cards.NewID(6).WithFace(6),
					Amount: 1,
					Number: "2",
				},
			},
		},
		{
			name: "collector without collection",
			filter: cards.NewFilter().
				WithName("ummy Card").
				WithCollector(cards.NewCollector("differentCollector")),
			expected: []cards.Card{
				{
					Name:   "Dummy Card 1",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard1.png"},
					ID:     cards.NewID(1).WithFace(1),
					Amount: 0,
					Number: "1",
				},
				{
					Name:   "Dummy Card 2",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard2.png"},
					ID:     cards.NewID(2).WithFace(2),
					Amount: 0,
					Number: "2",
				},
				{
					Name:   "Dummy Card 3",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard3.png"},
					ID:     cards.NewID(3).WithFace(3),
					Amount: 0,
					Number: "3",
				},
			},
		},
		{
			name: "only collected with name filter",
			filter: cards.NewFilter().
				WithName("ummy Card").
				WithCollector(collector).
				WithOnlyCollected(),
			expected: []cards.Card{
				{
					Name:   "Dummy Card 1",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard1.png"},
					ID:     cards.NewID(1).WithFace(1),
					Amount: 3,
					Number: "1",
				},
				{
					Name:   "Dummy Card 2",
					Set:    cards.Set{Code: "M10", Name: "Magic 2010"},
					Image:  cards.Image{URL: "http://localhost/images/dummyCard2.png"},
					ID:     cards.NewID(2).WithFace(2),
					Amount: 1,
					Number: "2",
				},
			},
		},
		{
			name: "all collected",
			filter: cards.NewFilter().
				WithCollector(collector).
				WithOnlyCollected(),
			expected: []cards.Card{
				{
					Name:   "Back Face doubleFace",
					Set:    cards.Set{Code: "M13", Name: "Magic 2013"},
					Image:  cards.Image{URL: "http://localhost/images/BackFace.png"},
					ID:     cards.NewID(8).WithFace(9),
					Amount: 2,
					Number: "1",
				},
				{
					Name:   "Card 11 with hash",
					Set:    cards.Set{Code: "M15", Name: "Magic 2015"},
					Image:  cards.Image{URL: "http://localhost/images/card11Hash.png"},
					ID:     cards.NewID(11).WithFace(12),
					Amount: 3,
					Number: "1",
				},
				{
					Name:   "Card 12 with hash",
					Set:    cards.Set{Code: "M15", Name: "Magic 2015"},
					Image:  cards.Image{URL: "http://localhost/images/card12hash.png"},
					ID:     cards.NewID(12).WithFace(13),
					Amount: 1,
					Number: "2",
				},
			},
		},
		{
			name: "only collected with name filter but empty collection",
			filter: cards.NewFilter().
				WithName("ummy Card").
				WithCollector(cards.NewCollector("differentCollector")).
				WithOnlyCollected(),
			expected: []cards.Card{},
		},
		{
			name: "all collected but empty collection",
			filter: cards.NewFilter().
				WithCollector(cards.NewCollector("differentCollector")).
				WithOnlyCollected(),
			expected: []cards.Card{},
		},
		{
			name: "all collected with collection and unknown cards",
			filter: cards.NewFilter().
				WithCollector(cards.NewCollector("myOtherUser")).
				WithOnlyCollected().
				WithLanguage(cards.DefaultLang),
			expected: []cards.Card{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := cardRepo.Find(ctx, tc.filter, cards.NewPage(1, 3))

			require.NoError(t, err)
			assert.ElementsMatch(t, tc.expected, result.Result)
		})
	}
}

func TestPrints(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	cardRepo := postgres.NewCardRepository(connection, cfg)

	cases := []struct {
		name      string
		cardName  string
		collector cards.Collector
		page      cards.Page
		expected  []cards.CardPrint
	}{
		{
			name:     "empty when empty name",
			cardName: "",
			page:     cards.DefaultPage(),
			expected: []cards.CardPrint{},
		},
		{
			name:     "empty when space only name",
			cardName: " ",
			page:     cards.DefaultPage(),
			expected: []cards.CardPrint{},
		},
		{
			name:     "match name on first page",
			cardName: "Print Card",
			page:     cards.NewPage(1, 2),
			expected: []cards.CardPrint{
				{
					Name:   "Print Card",
					Code:   "M16",
					ID:     cards.NewID(18).WithFace(19),
					Number: "31",
				},
				{
					Name:   "Print Card",
					Code:   "M15",
					ID:     cards.NewID(19).WithFace(20),
					Number: "32",
				},
			},
		},
		{
			name:     "match name on last page",
			cardName: "Print Card",
			page:     cards.NewPage(2, 2),
			expected: []cards.CardPrint{
				{
					Name:   "Print Card",
					Code:   "M14",
					ID:     cards.NewID(20).WithFace(21),
					Number: "33",
				},
			},
		},
		{
			name:     "empty prints when name not found",
			cardName: "Print Card Other",
			page:     cards.NewPage(1, 2),
			expected: []cards.CardPrint{},
		},
		{
			name:      "match name on first page with collector",
			cardName:  "Print Card",
			page:      cards.NewPage(1, 2),
			collector: collector,
			expected: []cards.CardPrint{
				{
					Name:   "Print Card",
					Code:   "M16",
					ID:     cards.NewID(18).WithFace(19),
					Number: "31",
					Amount: 3,
				},
				{
					Name:   "Print Card",
					Code:   "M15",
					ID:     cards.NewID(19).WithFace(20),
					Number: "32",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := cardRepo.Prints(ctx, tc.cardName, tc.collector, tc.page)

			require.NoError(t, err)
			require.Len(t, result.Result, len(tc.expected))
			assert.Equal(t, tc.page.Page(), result.Page)
			assert.ElementsMatch(t, tc.expected, result.Result)
		})
	}
}

func TestFindByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	repo := postgres.NewCollectionRepository(connection, cfg)

	ctx := context.Background()
	exist, err := repo.Exist(ctx, cards.NewID(1))

	require.NoError(t, err)
	assert.True(t, exist)
}

func TestFindByNoneExistingID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	repo := postgres.NewCollectionRepository(connection, cfg)

	ctx := context.Background()
	exist, err := repo.Exist(ctx, cards.NewID(1000))

	require.NoError(t, err)
	assert.False(t, exist)
}

func TestAddCards(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	repo := postgres.NewCollectionRepository(connection, cfg)
	item, err := cards.NewCollectable(cards.NewID(9), 2)
	require.NoError(t, err)

	ctx := context.Background()
	err = repo.Collect(ctx, item, collector)
	require.NoError(t, err)

	filter := cards.NewFilter().
		WithName("Uncollected Card 1").
		WithCollector(collector).
		WithOnlyCollected().
		WithLanguage(cards.DefaultLang)
	page, err := repo.Find(ctx, filter, cards.NewPage(1, 10))
	require.NoError(t, err)

	require.Len(t, page.Result, 1)
	assert.Truef(t, cards.NewID(9).Eq(page.Result[0].ID), "want card ID 9 got %d", page.Result[0].ID)
	assert.Equal(t, 2, page.Result[0].Amount)
}

func TestAddNoneExistingCardNoError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	repo := postgres.NewCollectionRepository(connection, cfg)
	noneExistingItem, _ := cards.NewCollectable(cards.NewID(1000), 1)

	ctx := context.Background()
	err := repo.Collect(ctx, noneExistingItem, collector)

	require.NoError(t, err)
}

func TestRemoveCards(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	repo := postgres.NewCollectionRepository(connection, cfg)
	item, err := cards.NewCollectable(cards.NewID(10), 0)
	require.NoError(t, err)

	ctx := context.Background()
	err = repo.Remove(ctx, item, collector)
	require.NoError(t, err)

	filter := cards.NewFilter().
		WithName("Remove Collected Card 1").
		WithCollector(collector).
		WithOnlyCollected().
		WithLanguage(cards.DefaultLang)
	page, err := repo.Find(ctx, filter, cards.NewPage(1, 10))
	require.NoError(t, err)

	require.Empty(t, page.Result)
}

func TestRemoveUncollectedCardNoError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	repo := postgres.NewCollectionRepository(connection, cfg)
	noneExistingItem, _ := cards.NewCollectable(cards.NewID(2000), 0)

	ctx := context.Background()
	err := repo.Remove(ctx, noneExistingItem, collector)

	require.NoError(t, err)
}
