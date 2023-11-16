package application_test

import (
	"fmt"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/collection/adapter/mocks"
	"github.com/konstantinfoerster/card-service-go/internal/collection/application"
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cardRepo = mocks.MockCardRepository{
	MockByID: func(id int) (*domain.Card, error) {
		if id == 1 {
			return &domain.Card{ID: 1}, nil
		}

		return nil, domain.ErrCardNotFound
	},
}

func TestCollectItem(t *testing.T) {
	c := domain.Collector{ID: "myUser"}
	i, err := domain.NewItem(1)
	require.NoError(t, err)
	collectionRepo := mocks.MockCollectionRepository{
		MockAdd: func(item domain.Item, collector domain.Collector) error {
			if item.ID != i.ID && collector.ID != c.ID {
				return fmt.Errorf("exptcted item %d and collector %s, but got %d and %s", i.ID, c.ID, item.ID, collector.ID)
			}

			return nil
		},
		MockCount: func(itemID int, collector domain.Collector) (int, error) {
			if itemID != i.ID && collector.ID != c.ID {
				return 0, fmt.Errorf("exptcted item %d and collector %s, but got %d and %s", i.ID, c.ID, itemID, collector.ID)
			}

			return 2, nil
		},
	}
	svc := application.NewCollectService(&collectionRepo, &cardRepo)

	collect, err := svc.Collect(i, c)

	require.NoError(t, err)
	assert.Equal(t, i.ID, collect.ID)
	assert.Equal(t, 2, collect.Amount)
}

func TestCollectNoneExistingItem(t *testing.T) {
	c := domain.Collector{ID: "myUser"}
	noneExistingItem, err := domain.NewItem(1000)
	require.NoError(t, err)
	svc := application.NewCollectService(nil, &cardRepo)

	collect, err := svc.Collect(noneExistingItem, c)

	assert.Equal(t, domain.CollectableResult{}, collect)
	var appErr common.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, common.ErrTypeInvalidInput, appErr.ErrorType)
}
