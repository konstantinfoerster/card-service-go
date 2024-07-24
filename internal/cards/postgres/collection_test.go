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

func TestFindByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	collectionRepo := postgres.NewCollectionRepository(connection, cfg)

	ctx := context.Background()
	result, err := collectionRepo.ByID(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, "Dummy Card 1", result.Name)
}

func TestFindByNoneExistingID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	collectionRepo := postgres.NewCollectionRepository(connection, cfg)

	ctx := context.Background()
	_, err := collectionRepo.ByID(ctx, 1000)

	require.ErrorIs(t, err, cards.ErrCardNotFound)
}

func TestFindCollectedByName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{Host: "http://localhost/"}
	collectionRepo := postgres.NewCollectionRepository(connection, cfg)

	ctx := context.Background()
	result, err := collectionRepo.FindCollectedByName(ctx, "ummy Card", collector, cards.NewPage(1, 10))

	require.NoError(t, err)
	require.Len(t, result.Result, 2)
	assert.Equal(t, "Dummy Card 1", result.Result[0].Name)
	assert.Equal(t, "http://localhost/images/dummyCard1.png", result.Result[0].Image)
	assert.Equal(t, 3, result.Result[0].Amount)
	assert.Equal(t, "Dummy Card 2", result.Result[1].Name)
	assert.Equal(t, "http://localhost/images/dummyCard2.png", result.Result[1].Image)
	assert.Equal(t, 1, result.Result[1].Amount)
}

func TestAddCards(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	collectionRepo := postgres.NewCollectionRepository(connection, cfg)
	item, err := cards.NewCollectable(9, 2)
	require.NoError(t, err)

	ctx := context.Background()
	err = collectionRepo.Upsert(ctx, item, collector)
	require.NoError(t, err)

	page, err := collectionRepo.FindCollectedByName(ctx, "Uncollected Card 1", collector, cards.NewPage(1, 10))
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
	collectionRepo := postgres.NewCollectionRepository(connection, cfg)
	noneExistingItem, _ := cards.NewCollectable(1000, 1)

	ctx := context.Background()
	err := collectionRepo.Upsert(ctx, noneExistingItem, collector)

	require.NoError(t, err)
}

func TestRemoveCards(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	collectionRepo := postgres.NewCollectionRepository(connection, cfg)
	item, err := cards.RemoveItem(10)
	require.NoError(t, err)

	ctx := context.Background()
	err = collectionRepo.Remove(ctx, item, collector)
	require.NoError(t, err)

	page, err := collectionRepo.FindCollectedByName(ctx, "Remove Collected Card 1", collector, cards.NewPage(1, 10))
	require.NoError(t, err)

	require.Empty(t, page.Result)
}

func TestRemoveUncollectedCardNoError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	collectionRepo := postgres.NewCollectionRepository(connection, cfg)
	noneExistingItem, _ := cards.RemoveItem(2000)

	ctx := context.Background()
	err := collectionRepo.Remove(ctx, noneExistingItem, collector)

	require.NoError(t, err)
}
