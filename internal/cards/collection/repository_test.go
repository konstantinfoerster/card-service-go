package collection_test

import (
	"context"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/collection"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var repository collection.Repository

var collector = cards.NewCollector("myUser")

func TestIntegrationCollectionRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := config.Images{
		Host: "http://localhost/",
	}
	runner := test.NewRunner()
	runner.Run(t, func(t *testing.T) {
		repository = collection.NewRepository(runner.Connection(), cfg)

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
	result, err := repository.ByID(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, "Dummy Card 1", result.Name)
}

func findByNoneExistingID(t *testing.T) {
	ctx := context.Background()
	_, err := repository.ByID(ctx, 1000)

	require.ErrorIs(t, err, cards.ErrCardNotFound)
}

func findCollectedByName(t *testing.T) {
	ctx := context.Background()
	result, err := repository.FindCollectedByName(ctx, "ummy Card", collector, common.NewPage(1, 10))

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
	item, err := collection.NewItem(9, 2)
	require.NoError(t, err)

	err = repository.Upsert(ctx, item, collector)
	require.NoError(t, err)

	page, err := repository.FindCollectedByName(ctx, "Uncollected Card 1", collector, common.NewPage(1, 10))
	require.NoError(t, err)

	require.Len(t, page.Result, 1)
	assert.Equal(t, 9, page.Result[0].ID)
	assert.Equal(t, 2, page.Result[0].Amount)
}

func addNoneExistingCardNoError(t *testing.T) {
	ctx := context.Background()
	noneExistingItem, _ := collection.NewItem(1000, 1)

	err := repository.Upsert(ctx, noneExistingItem, collector)

	require.NoError(t, err)
}

func removeCards(t *testing.T) {
	ctx := context.Background()
	item, err := collection.RemoveItem(10)
	require.NoError(t, err)

	err = repository.Remove(ctx, item, collector)
	require.NoError(t, err)

	page, err := repository.FindCollectedByName(ctx, "Remove Collected Card 1", collector, common.NewPage(1, 10))
	require.NoError(t, err)

	require.Empty(t, page.Result)
}

func removeUncollectedCardNoError(t *testing.T) {
	ctx := context.Background()
	noneExistingItem, _ := collection.RemoveItem(2000)

	err := repository.Remove(ctx, noneExistingItem, collector)

	require.NoError(t, err)
}
