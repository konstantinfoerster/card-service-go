package memory_test

import (
	"fmt"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detection"
	"github.com/konstantinfoerster/card-service-go/internal/cards/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTop5MatchesByHash(t *testing.T) {
	images := []detection.ImageHash{
		imageHash(1, "Card 1", 30),
		imageHash(2, "Card 2", 10),
		imageHash(3, "Card 3", 20),
		imageHash(4, "Card 4", detection.MaxScore),
	}
	expected := cards.Scores{
		{ID: cards.NewID(2).WithFace(2), Score: 10},
		{ID: cards.NewID(3).WithFace(3), Score: 20},
		{ID: cards.NewID(1).WithFace(1), Score: 30},
	}
	repo := memory.NewDetectRepository(images)

	result, err := repo.Top5MatchesByHash(t.Context(), hashWithDistance(0))

	require.NoError(t, err)
	assert.Equal(t, expected, result, "best first, without scores of MaxScore and more")
}

func TestTop5MatchesByHash_BestHashCounts(t *testing.T) {
	images := []detection.ImageHash{imageHash(1, "Card 1", 50)}
	expected := cards.Scores{{ID: cards.NewID(1).WithFace(1), Score: 10}}
	repo := memory.NewDetectRepository(images)

	result, err := repo.Top5MatchesByHash(t.Context(), hashWithDistance(0), hashWithDistance(40))

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTop5MatchesByHash_OnlyBestMatchPerName(t *testing.T) {
	images := []detection.ImageHash{
		imageHash(1, "Same Name", 20),
		imageHash(2, "Same Name", 10),
		imageHash(3, "Same Name", 10),
	}
	expected := cards.Scores{{ID: cards.NewID(2).WithFace(2), Score: 10}}

	result, err := memory.NewDetectRepository(images).Top5MatchesByHash(t.Context(), hashWithDistance(0))

	require.NoError(t, err)
	assert.Equal(t, expected, result, "the lowest id wins on equal scores")
}

func TestTop5MatchesByHash_AtMostFive(t *testing.T) {
	images := make([]detection.ImageHash, 0, 6)
	for i := 1; i <= 6; i++ {
		images = append(images, imageHash(i, fmt.Sprintf("Card %d", i), i))
	}

	result, err := memory.NewDetectRepository(images).Top5MatchesByHash(t.Context(), hashWithDistance(0))

	require.NoError(t, err)
	assert.Len(t, result, 5)
}

func TestTop5MatchesByHash_NoHashes(t *testing.T) {
	images := []detection.ImageHash{imageHash(1, "Card 1", 0)}

	result, err := memory.NewDetectRepository(images).Top5MatchesByHash(t.Context())

	require.NoError(t, err)
	assert.Empty(t, result)
}

// imageHash returns an image of the given card with a hash of the given distance to the zero hash.
func imageHash(id int, name string, distance int) detection.ImageHash {
	return detection.ImageHash{
		ID:   cards.NewID(id).WithFace(id),
		Name: name,
		Hash: hashWithDistance(distance),
	}
}

// hashWithDistance returns a hash with the given distance to the zero hash, the first bits are set.
func hashWithDistance(distance int) detection.Hash {
	words := make([]uint64, 13)
	for i := range distance {
		words[i/64] |= 1 << (i % 64)
	}

	return detection.Hash{PHashR: words[0:4], PHashG: words[4:8], PHashB: words[8:12], DHash: words[12]}
}
