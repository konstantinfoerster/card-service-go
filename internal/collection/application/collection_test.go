package application_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/collection/adapter/fakes"
	"github.com/konstantinfoerster/card-service-go/internal/collection/application"
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/img"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectItem(t *testing.T) {
	svc := newCollectionService(t)
	c := domain.Collector{ID: "myUser"}
	item, err := domain.NewItem(1, 2)
	require.NoError(t, err)

	collect, err := svc.Collect(item, c)

	require.NoError(t, err)
	assert.Equal(t, item.ID, collect.ID)
	assert.Equal(t, item.Amount, collect.Amount)
}

func TestCollectNoneExistingItem(t *testing.T) {
	svc := newCollectionService(t)
	c := domain.Collector{ID: "myUser"}
	noneExistingItem, err := domain.NewItem(1000, 0)
	require.NoError(t, err)

	collect, err := svc.Collect(noneExistingItem, c)

	assert.Equal(t, domain.Item{}, collect)
	var appErr common.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, common.ErrTypeInvalidInput, appErr.ErrorType)
}

func newCollectionService(t *testing.T) application.CollectionService {
	repo, err := fakes.NewRepository(img.NewPHasher())
	require.NoError(t, err)

	return application.NewCollectionService(repo, repo)
}
