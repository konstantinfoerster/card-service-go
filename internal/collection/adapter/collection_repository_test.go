package adapter_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/collection/adapter"
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var collectionRepo domain.CollectionRepository

var noneExistingItem, _ = domain.NewItem(1000)

func TestIntegrationCollectionRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	runner := commontest.NewRunner()
	runner.Run(t, func(t *testing.T) {
		cfg := config.Images{
			Host: "http://localhost",
		}
		collectionRepo = adapter.NewCollectionRepository(runner.Connection(), cfg)

		t.Run("find collected by name", findCollectedByName)
		t.Run("add cards to collection", addCards)
		t.Run("add none existing card", addNoneExistingCard)
		t.Run("remove cards from collection", removeCards)
		t.Run("remove uncollected", removeUncollectedCard)
		t.Run("count uncollected card", countNoneCollectedCard)
	})
}

func findCollectedByName(t *testing.T) {
	c := domain.Collector{ID: "myUser"}
	result, err := collectionRepo.FindByName("ummy Card", domain.NewPage(1, 10), c)

	require.NoError(t, err)
	assert.Len(t, result.Result, 2)
	assert.Equal(t, "Dummy Card 1", result.Result[0].Name)
	assert.Equal(t, "http://localhost/images/dummyCard1.png", result.Result[0].Image)
	assert.Equal(t, 3, result.Result[0].Amount)
	assert.Equal(t, "Dummy Card 2", result.Result[1].Name)
	assert.Equal(t, "http://localhost/images/dummyCard2.png", result.Result[1].Image)
	assert.Equal(t, 1, result.Result[1].Amount)
}

func addCards(t *testing.T) {
	c := domain.Collector{ID: "myUser"}
	item, err := domain.NewItem(9)
	require.NoError(t, err)

	err = collectionRepo.Add(item, c)
	require.NoError(t, err)
	err = collectionRepo.Add(item, c)
	require.NoError(t, err)

	count, err := collectionRepo.Count(item.ID, c)
	require.NoError(t, err)

	assert.Equal(t, 2, count)
}

func addNoneExistingCard(t *testing.T) {
	c := domain.Collector{ID: "myUser"}

	err := collectionRepo.Add(noneExistingItem, c)

	require.NoError(t, err)
}

func removeCards(t *testing.T) {
	c := domain.Collector{ID: "myUser"}
	itemID := 10

	err := collectionRepo.Remove(itemID, c)
	require.NoError(t, err)
	err = collectionRepo.Remove(itemID, c)
	require.NoError(t, err)

	count, err := collectionRepo.Count(itemID, c)
	require.NoError(t, err)

	assert.Empty(t, count)
}

func removeUncollectedCard(t *testing.T) {
	c := domain.Collector{ID: "myUser"}
	err := collectionRepo.Remove(noneExistingItem.ID, c)

	require.NoError(t, err)
}

func countNoneCollectedCard(t *testing.T) {
	c := domain.Collector{ID: "myUser"}

	count, err := collectionRepo.Count(0, c)

	require.NoError(t, err)
	assert.Empty(t, count)
}
