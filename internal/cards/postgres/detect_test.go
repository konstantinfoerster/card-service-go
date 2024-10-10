package postgres_test

import (
	"context"
	"testing"

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
	result, err := detectRepo.Top5MatchesByHash(ctx, unknownHash, hash, unknownHash)

	require.NoError(t, err)
	require.Len(t, result, 3)
	for _, r := range result {
		assert.Less(t, r.Score, 60)
		assert.Positive(t, r.Score)
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
	result, err := detectRepo.Top5MatchesByHash(ctx, unknownHash)

	require.NoError(t, err)
	require.Empty(t, result)
}
