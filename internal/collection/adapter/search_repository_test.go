package adapter_test

import (
	"context"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/collection/adapter"
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	"github.com/konstantinfoerster/card-service-go/internal/common/postgres"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var serchRepository domain.SearchRepository

func TestIntegrationCardRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	runner := commontest.NewRunner()
	runner.Run(t, func(t *testing.T) {
		serchRepository = newCardRepository(t, runner.Connection())

		t.Run("find by id", findByID)
		t.Run("find by id with none existing id", findByNoneExistingID)

		t.Run("find by name default page", findByName)
		t.Run("find by name last page", findByNameLastPage)
		t.Run("find by name double face card", findByNameDoubleFace)
		t.Run("find by empty or unknown term", findByNameNoResult)
		t.Run("find by card with no image", findByNameNoImageURL)

		t.Run("find by name and collector", findByNameAndCollector)
		t.Run("find by name and collector double face card", findByNameAndCollectorDoubleFace)
		t.Run("find by name and collector with no image", findByNameAndCollectorNoImageURL)

		t.Run("top 5 matches by hash", top5MatchesByHash)
		t.Run("top 5 matches by hash no result", top5MatchesByHashNoResult)

		t.Run("top 5 matches by collector and hash", top5MatchesByCollectorAndHash)
		t.Run("top 5 matches by collector and hash no result", top5MatchesByCollectorAndHashNoResult)
	})
}

func findByID(t *testing.T) {
	result, err := serchRepository.ByID(1)

	require.NoError(t, err)
	assert.Equal(t, "Dummy Card 1", result.Name)
}

func findByNoneExistingID(t *testing.T) {
	_, err := serchRepository.ByID(1000)

	require.ErrorIs(t, err, domain.ErrCardNotFound)
}

func findByName(t *testing.T) {
	result, err := serchRepository.FindByName("ummy Card", common.NewPage(1, 3))

	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Len(t, result.Result, 3)
	assert.True(t, result.HasMore)
	assert.Equal(t, "Dummy Card 1", result.Result[0].Name)
	assert.Equal(t, "http://localhost/images/dummyCard1.png", result.Result[0].Image)
	assert.Equal(t, "Dummy Card 2", result.Result[1].Name)
	assert.Equal(t, "http://localhost/images/dummyCard2.png", result.Result[1].Image)
	assert.Equal(t, "Dummy Card 3", result.Result[2].Name)
	assert.Equal(t, "http://localhost/images/dummyCard3.png", result.Result[2].Image)
}

func findByNameLastPage(t *testing.T) {
	result, err := serchRepository.FindByName("Dummy Card", common.NewPage(2, 3))

	require.NoError(t, err)
	assert.False(t, result.HasMore)
	assert.Equal(t, 2, result.Page)
	assert.Len(t, result.Result, 1)
	assert.Equal(t, "http://localhost/images/dummyCard4.png", result.Result[0].Image)
}

func findByNameDoubleFace(t *testing.T) {
	cases := []struct {
		name       string
		searchTerm string
		resultSize int
	}{
		{
			name:       "card name",
			searchTerm: "Double Face",
			resultSize: 0,
		},
		{
			name:       "front face name",
			searchTerm: "Front face ",
			resultSize: 1,
		},
		{
			name:       "back face name",
			searchTerm: "Back Face",
			resultSize: 1,
		},
		{
			name:       "both faces",
			searchTerm: "doubleFace",
			resultSize: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := serchRepository.FindByName(tc.searchTerm, common.NewPage(1, 10))

			require.NoError(t, err)
			assert.Len(t, result.Result, tc.resultSize)
		})
	}
}

func findByNameNoImageURL(t *testing.T) {
	result, err := serchRepository.FindByName("No Image Card", common.NewPage(1, 5))

	require.NoError(t, err)
	assert.Len(t, result.Result, 2)
	assert.Equal(t, "", result.Result[0].Image)
	assert.Equal(t, "http://localhost/images/noFace.png", result.Result[1].Image)
}

func findByNameNoResult(t *testing.T) {
	cases := []struct {
		name       string
		searchTerm string
	}{
		{
			name:       "Empty term",
			searchTerm: "",
		},
		{
			name:       "Space only term",
			searchTerm: " ",
		},
		{
			name:       "unknown term",
			searchTerm: "DoesNotExists",
		},
		{
			name:       "different language",
			searchTerm: "French",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := serchRepository.FindByName(tc.searchTerm, common.NewPage(1, 10))

			require.NoError(t, err)
			assert.Equal(t, 1, result.Page)
			assert.False(t, result.HasMore)
			assert.Empty(t, result.Result)
		})
	}
}

func findByNameAndCollector(t *testing.T) {
	c := domain.Collector{ID: "myUser"}
	result, err := serchRepository.FindByCollectorAndName(c, "ummy Card", common.NewPage(1, 3))

	require.NoError(t, err)
	assert.Len(t, result.Result, 3)
	assert.Equal(t, "Dummy Card 1", result.Result[0].Name)
	assert.Equal(t, "http://localhost/images/dummyCard1.png", result.Result[0].Image)
	assert.Equal(t, 3, result.Result[0].Amount)
	assert.Equal(t, "Dummy Card 2", result.Result[1].Name)
	assert.Equal(t, "http://localhost/images/dummyCard2.png", result.Result[1].Image)
	assert.Equal(t, 1, result.Result[1].Amount)
	assert.Equal(t, "Dummy Card 3", result.Result[2].Name)
	assert.Equal(t, "http://localhost/images/dummyCard3.png", result.Result[2].Image)
	assert.Empty(t, result.Result[2].Amount)
}

func findByNameAndCollectorDoubleFace(t *testing.T) {
	cases := []struct {
		name       string
		searchTerm string
		resultSize int
		amount     int
	}{
		{
			name:       "card name",
			searchTerm: "Double Face",
			resultSize: 0,
			amount:     2,
		},
		{
			name:       "front face name",
			searchTerm: "Front face ",
			resultSize: 1,
			amount:     2,
		},
		{
			name:       "back face name",
			searchTerm: "Back Face",
			resultSize: 1,
			amount:     2,
		},
		{
			name:       "both faces",
			searchTerm: "doubleFace",
			resultSize: 2,
			amount:     2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := domain.Collector{ID: "myUser"}
			result, err := serchRepository.FindByCollectorAndName(c, tc.searchTerm, common.NewPage(1, 10))

			require.NoError(t, err)
			assert.Len(t, result.Result, tc.resultSize)

			for _, r := range result.Result {
				assert.Equal(t, tc.amount, r.Amount)
			}
		})
	}
}

func findByNameAndCollectorNoImageURL(t *testing.T) {
	c := domain.Collector{ID: "myUser"}
	result, err := serchRepository.FindByCollectorAndName(c, "No Image Card", common.NewPage(1, 10))

	require.NoError(t, err)
	assert.Len(t, result.Result, 2)
	assert.Equal(t, 5, result.Result[0].Amount)
	assert.Equal(t, "", result.Result[0].Image)
	assert.Equal(t, 1, result.Result[1].Amount)
	assert.Equal(t, "http://localhost/images/noFace.png", result.Result[1].Image)
}

func top5MatchesByHash(t *testing.T) {
	ctx := context.Background()
	unknownHash := domain.Hash{Value: []uint64{1, 2, 3, 4}}
	hash := domain.Hash{
		Value: []uint64{
			9223372036854775807,
			8828676655832293646,
			8002350605550622951,
			4369376647429299945,
		},
	}

	result, err := serchRepository.Top5MatchesByHash(ctx, unknownHash, hash, unknownHash)

	require.NoError(t, err)
	require.Len(t, result, 3)
	for _, r := range result {
		assert.Less(t, r.Score, 60)
		assert.Contains(t, r.Name, "with hash")
		assert.Empty(t, r.Amount)
	}
}

func top5MatchesByHashNoResult(t *testing.T) {
	ctx := context.Background()
	unknownHash := domain.Hash{Value: []uint64{1, 2, 3, 4}}

	result, err := serchRepository.Top5MatchesByHash(ctx, unknownHash)

	require.NoError(t, err)
	require.Empty(t, result)
}

func top5MatchesByCollectorAndHash(t *testing.T) {
	ctx := context.Background()
	c := domain.Collector{ID: "myUser"}
	unknownHash := domain.Hash{Value: []uint64{1, 2, 3, 4}}
	hash := domain.Hash{
		Value: []uint64{
			9223372036854775807,
			8828676655832293646,
			8002350605550622951,
			4369376647429299945,
		},
	}

	result, err := serchRepository.Top5MatchesByCollectorAndHash(ctx, c, unknownHash, hash, unknownHash)

	require.NoError(t, err)
	require.Len(t, result, 3)
	for _, r := range result {
		assert.Less(t, r.Score, 60)
		assert.Contains(t, r.Name, "with hash")
	}
	assert.NotEmpty(t, result[0].Amount)
	assert.NotEmpty(t, result[1].Amount)
	assert.Empty(t, result[2].Amount)
}

func top5MatchesByCollectorAndHashNoResult(t *testing.T) {
	ctx := context.Background()
	c := domain.Collector{ID: "myUser"}
	unknownHash := domain.Hash{Value: []uint64{1, 2, 3, 4}}

	result, err := serchRepository.Top5MatchesByCollectorAndHash(ctx, c, unknownHash)

	require.NoError(t, err)
	require.Empty(t, result)
}

func newCardRepository(t *testing.T, con *postgres.DBConnection) domain.SearchRepository {
	t.Helper()

	require.NotNil(t, con)

	cfg := config.Images{
		Host: "http://localhost",
	}

	return adapter.NewSearchRepository(con, cfg)
}
