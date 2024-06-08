package adapter_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/collection/adapter"
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var collectionRepo domain.CollectionRepository

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
		t.Run("add none existing card should work", addNoneExistingCardNoError)
		t.Run("remove cards from collection", removeCards)
		t.Run("remove uncollected should work", removeUncollectedCardNoError)
	})
}

func findCollectedByName(t *testing.T) {
	c := domain.Collector{ID: "myUser"}
	result, err := collectionRepo.FindCollectedByName("ummy Card", common.NewPage(1, 10), c)

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
	c := domain.Collector{ID: "myUser"}
	item, err := domain.NewItem(9, 2)
	require.NoError(t, err)

	err = collectionRepo.Upsert(item, c)
	require.NoError(t, err)

	page, err := collectionRepo.FindCollectedByName("Uncollected Card 1", common.NewPage(1, 10), c)
	require.NoError(t, err)

	require.Len(t, page.Result, 1)
	assert.Equal(t, 9, page.Result[0].ID)
	assert.Equal(t, 2, page.Result[0].Amount)
}

func addNoneExistingCardNoError(t *testing.T) {
	c := domain.Collector{ID: "myUser"}
	noneExistingItem, _ := domain.NewItem(1000, 1)

	err := collectionRepo.Upsert(noneExistingItem, c)

	require.NoError(t, err)
}

func removeCards(t *testing.T) {
	c := domain.Collector{ID: "myUser"}
	cardID := 10

	err := collectionRepo.Remove(cardID, c)
	require.NoError(t, err)

	page, err := collectionRepo.FindCollectedByName("Remove Collected Card 1", common.NewPage(1, 10), c)
	require.NoError(t, err)

	require.Empty(t, page.Result)
}

func removeUncollectedCardNoError(t *testing.T) {
	c := domain.Collector{ID: "myUser"}
	noneExistingItem, _ := domain.NewItem(2000, 1)
	err := collectionRepo.Remove(noneExistingItem.ID, c)

	require.NoError(t, err)
}
