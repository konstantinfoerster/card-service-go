package cards_test

import (
	"context"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/memory"
	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectItem(t *testing.T) {
	ctx := context.Background()
	svc := newCollectionService(t)
	item, err := cards.NewItem(1, 2)
	require.NoError(t, err)

	collect, err := svc.Collect(ctx, item, cards.NewCollector("myUser"))

	require.NoError(t, err)
	assert.Equal(t, item.ID, collect.ID)
	assert.Equal(t, item.Amount, collect.Amount)
}

func TestCollectNoneExistingItem(t *testing.T) {
	ctx := context.Background()
	svc := newCollectionService(t)
	noneExistingItem, err := cards.NewItem(1000, 1)
	require.NoError(t, err)

	collect, err := svc.Collect(ctx, noneExistingItem, cards.NewCollector("myUser"))

	assert.Equal(t, cards.Item{}, collect)
	var appErr aerrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, aerrors.ErrTypeInvalidInput, appErr.ErrorType)
}

func newCollectionService(t *testing.T) cards.CollectionService {
	seed, err := test.CardSeed()
	require.NoError(t, err)
	repo, err := memory.NewCollectionRepository(seed)
	require.NoError(t, err)

	return cards.NewCollectionService(repo)
}
