package postgres_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detection"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllHashes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	detectRepo := postgres.NewDetectRepository(connection)
	allBits := uint64(0xFFFFFFFFFFFFFFFF)
	phash := []uint64{allBits, allBits, allBits, allBits}
	expected := detection.ImageHash{
		ID:   cards.NewID(21).WithFace(22),
		Name: "Same Name Hash Card",
		Hash: detection.Hash{PHashR: phash, PHashG: phash, PHashB: phash, DHash: allBits},
	}

	result, err := detectRepo.AllHashes(t.Context())

	require.NoError(t, err)
	assert.Len(t, result, 6, "only images with hashes")
	assert.Contains(t, result, expected)
}
