package postgres_test

import (
	"context"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var collector = cards.NewCollector("myUser")
var collectionRepo cards.CollectionRepository

func TestIntegrationCollectionRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := config.Images{
		Host: "http://localhost/",
	}
	runner := test.NewRunner()
	runner.Run(t, func(t *testing.T) {
		collectionRepo = postgres.NewCollectionRepository(runner.Connection(), cfg)

		t.Run("find by id", findByID)
		t.Run("find by id with none existing id", findByNoneExistingID)
		t.Run("find collected by name", findCollectedByName)
		t.Run("add cards to collection", addCards)
		t.Run("add none existing card should work", addNoneExistingCardNoError)
		t.Run("remove cards from collection", removeCards)
		t.Run("remove uncollected should work", removeUncollectedCardNoError)
	})
}

func findByID(t *testing.T) {
	ctx := context.Background()
	result, err := collectionRepo.ByID(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, "Dummy Card 1", result.Name)
}

func findByNoneExistingID(t *testing.T) {
	ctx := context.Background()
	_, err := collectionRepo.ByID(ctx, 1000)

	require.ErrorIs(t, err, cards.ErrCardNotFound)
}

func findCollectedByName(t *testing.T) {
	ctx := context.Background()
	result, err := collectionRepo.FindCollectedByName(ctx, "ummy Card", collector, common.NewPage(1, 10))

	require.NoError(t, err)
	require.Len(t, result.Result, 2)
	assert.Equal(t, "Dummy Card 1", result.Result[0].Name)
	assert.Equal(t, "http://localhost/images/dummyCard1.png", result.Result[0].Image)
	assert.Equal(t, 3, result.Result[0].Amount)
	assert.Equal(t, "Dummy Card 2", result.Result[1].Name)
	assert.Equal(t, "http://localhost/images/dummyCard2.png", result.Result[1].Image)
	assert.Equal(t, 1, result.Result[1].Amount)
}

func addCards(t *testing.T) {
	ctx := context.Background()
	item, err := cards.NewItem(9, 2)
	require.NoError(t, err)

	err = collectionRepo.Upsert(ctx, item, collector)
	require.NoError(t, err)

	page, err := collectionRepo.FindCollectedByName(ctx, "Uncollected Card 1", collector, common.NewPage(1, 10))
	require.NoError(t, err)

	require.Len(t, page.Result, 1)
	assert.Equal(t, 9, page.Result[0].ID)
	assert.Equal(t, 2, page.Result[0].Amount)
}

func addNoneExistingCardNoError(t *testing.T) {
	ctx := context.Background()
	noneExistingItem, _ := cards.NewItem(1000, 1)

	err := collectionRepo.Upsert(ctx, noneExistingItem, collector)

	require.NoError(t, err)
}

func removeCards(t *testing.T) {
	ctx := context.Background()
	item, err := cards.RemoveItem(10)
	require.NoError(t, err)

	err = collectionRepo.Remove(ctx, item, collector)
	require.NoError(t, err)

	page, err := collectionRepo.FindCollectedByName(ctx, "Remove Collected Card 1", collector, common.NewPage(1, 10))
	require.NoError(t, err)

	require.Empty(t, page.Result)
}

func removeUncollectedCardNoError(t *testing.T) {
	ctx := context.Background()
	noneExistingItem, _ := cards.RemoveItem(2000)

	err := collectionRepo.Remove(ctx, noneExistingItem, collector)

	require.NoError(t, err)
}
