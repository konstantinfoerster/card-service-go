package detect_test

import (
	"context"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detect"
	"github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var repository detect.Repository

func TestIntegrationCardRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := config.Images{
		Host: "http://localhost/",
	}
	runner := test.NewRunner()
	runner.Run(t, func(t *testing.T) {
		repository = detect.NewRepository(runner.Connection(), cfg)

		t.Run("top 5 matches by hash", top5MatchesByHash)
		t.Run("top 5 matches by hash no result", top5MatchesByHashNoResult)

		t.Run("top 5 matches by collector and hash", top5MatchesByCollectorAndHash)
		t.Run("top 5 matches by collector and hash no result", top5MatchesByCollectorAndHashNoResult)
	})
}

func top5MatchesByHash(t *testing.T) {
	ctx := context.Background()
	unknownHash := detect.Hash{Value: []uint64{1, 2, 3, 4}}
	hash := detect.Hash{
		Value: []uint64{
			9223372036854775807,
			8828676655832293646,
			8002350605550622951,
			4369376647429299945,
		},
	}

	result, err := repository.Top5MatchesByHash(ctx, unknownHash, hash, unknownHash)

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
	unknownHash := detect.Hash{Value: []uint64{1, 2, 3, 4}}

	result, err := repository.Top5MatchesByHash(ctx, unknownHash)

	require.NoError(t, err)
	require.Empty(t, result)
}

func top5MatchesByCollectorAndHash(t *testing.T) {
	ctx := context.Background()
	c := cards.Collector{ID: "myUser"}
	unknownHash := detect.Hash{Value: []uint64{1, 2, 3, 4}}
	hash := detect.Hash{
		Value: []uint64{
			9223372036854775807,
			8828676655832293646,
			8002350605550622951,
			4369376647429299945,
		},
	}

	result, err := repository.Top5MatchesByCollectorAndHash(ctx, c, unknownHash, hash, unknownHash)

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
	c := cards.Collector{ID: "myUser"}
	unknownHash := detect.Hash{Value: []uint64{1, 2, 3, 4}}

	result, err := repository.Top5MatchesByCollectorAndHash(ctx, c, unknownHash)

	require.NoError(t, err)
	require.Empty(t, result)
}
