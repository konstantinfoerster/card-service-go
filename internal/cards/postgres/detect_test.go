package postgres_test

import (
	"context"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTop5MatchesByHash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	detectRepo := postgres.NewDetectRepository(connection, postgres.Images{})
	unknownHash := sameChannelHash([]uint64{1, 2, 3, 4}, 0)
	hash := sameChannelHash([]uint64{
		9223372036854775807,
		8828676655832293646,
		8002350605550622951,
		4369376647429299945,
	}, 0xA5A5A5A5A5A5A5A5)

	ctx := context.Background()
	result, err := detectRepo.Top5MatchesByHash(ctx, unknownHash, hash, unknownHash)

	require.NoError(t, err)
	require.Len(t, result, 3)
	for _, r := range result {
		assert.Less(t, r.Score, cards.PHashThreshold*3+cards.DHashThreshold)
		assert.Positive(t, r.Score)
	}
}

func TestTop5MatchesByHashNoResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg := postgres.Images{}
	detectRepo := postgres.NewDetectRepository(connection, cfg)
	unknownHash := sameChannelHash([]uint64{1, 2, 3, 4}, 0)

	ctx := context.Background()
	result, err := detectRepo.Top5MatchesByHash(ctx, unknownHash)

	require.NoError(t, err)
	require.Empty(t, result)
}

// sameChannelHash builds a Hash using the same value for all three phash channels and
// the given dhash.
func sameChannelHash(phash []uint64, dhash uint64) cards.Hash {
	return cards.Hash{PHashR: phash, PHashG: phash, PHashB: phash, DHash: dhash}
}
