package collection_test

import (
	"context"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/collection"
	"github.com/konstantinfoerster/card-service-go/internal/cards/fakes"
	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectItem(t *testing.T) {
	svc := newCollectionService(t)
	ctx := context.Background()
	item, err := collection.NewItem(1, 2)
	require.NoError(t, err)

	collect, err := svc.Collect(ctx, item, cards.NewCollector("myUser"))

	require.NoError(t, err)
	assert.Equal(t, item.ID, collect.ID)
	assert.Equal(t, item.Amount, collect.Amount)
}

func TestCollectNoneExistingItem(t *testing.T) {
	ctx := context.Background()
	svc := newCollectionService(t)
	noneExistingItem, err := collection.NewItem(1000, 1)
	require.NoError(t, err)

	collect, err := svc.Collect(ctx, noneExistingItem, cards.NewCollector("myUser"))

	assert.Equal(t, collection.Item{}, collect)
	var appErr aerrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, aerrors.ErrTypeInvalidInput, appErr.ErrorType)
}

func newCollectionService(t *testing.T) collection.Service {
	repo, err := fakes.NewRepository(nil)
	require.NoError(t, err)

	return collection.NewService(repo)
}
