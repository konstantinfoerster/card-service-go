package postgres_test

import (
	"context"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTop5MatchesByHash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	detectRepo := postgres.NewDetectRepository(connection, config.Images{})
	unknownHash := image.Hash{Value: []uint64{1, 2, 3, 4}}
	hash := image.Hash{
		Value: []uint64{
			9223372036854775807,
			8828676655832293646,
			8002350605550622951,
			4369376647429299945,
		},
	}

	ctx := context.Background()
	result, err := detectRepo.Top5MatchesByHash(ctx, cards.Collector{}, unknownHash, hash, unknownHash)

	require.NoError(t, err)
	require.Len(t, result, 3)
	for _, r := range result {
		assert.Less(t, r.Score, 60)
		assert.Contains(t, r.Name, "with hash")
		assert.Empty(t, r.Amount)
	}
}

func Top5MatchesByHashNoResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	detectRepo := postgres.NewDetectRepository(connection, cfg)
	unknownHash := image.Hash{Value: []uint64{1, 2, 3, 4}}

	ctx := context.Background()
	result, err := detectRepo.Top5MatchesByHash(ctx, cards.Collector{}, unknownHash)

	require.NoError(t, err)
	require.Empty(t, result)
}

func Top5MatchesByCollectorAndHash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	detectRepo := postgres.NewDetectRepository(connection, cfg)
	unknownHash := image.Hash{Value: []uint64{1, 2, 3, 4}}
	hash := image.Hash{
		Value: []uint64{
			9223372036854775807,
			8828676655832293646,
			8002350605550622951,
			4369376647429299945,
		},
	}

	ctx := context.Background()
	result, err := detectRepo.Top5MatchesByHash(ctx, collector, unknownHash, hash, unknownHash)

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

func Top5MatchesByCollectorAndHashNoResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := config.Images{}
	detectRepo := postgres.NewDetectRepository(connection, cfg)
	unknownHash := image.Hash{Value: []uint64{1, 2, 3, 4}}

	ctx := context.Background()
	result, err := detectRepo.Top5MatchesByHash(ctx, collector, unknownHash)

	require.NoError(t, err)
	require.Empty(t, result)
}
