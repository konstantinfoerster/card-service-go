package detection_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingFinder struct {
	hashes []detection.Hash
	called bool
	scores cards.Scores
	err    error
}

func (f *recordingFinder) Top5MatchesByHash(_ context.Context, hashes ...detection.Hash) (cards.Scores, error) {
	f.called = true
	f.hashes = hashes

	return f.scores, f.err
}

func TestTop5MatchesHashesBothOrientations(t *testing.T) {
	img := openImage(t, "cardImage.jpg")
	finder := &recordingFinder{}

	_, err := detection.NewMatcher(finder, detection.NewHasher()).Top5Matches(t.Context(), img)

	require.NoError(t, err)
	require.Len(t, finder.hashes, 2)
	assert.NotEqual(t, finder.hashes[0], finder.hashes[1])
}

func TestTop5MatchesReturnsFinderScores(t *testing.T) {
	img := openImage(t, "cardImage.jpg")
	expected := cards.Scores{
		{ID: cards.NewID(1).WithFace(1), Score: 4},
	}
	finder := &recordingFinder{
		scores: expected}

	result, err := detection.NewMatcher(finder, detection.NewHasher()).Top5Matches(t.Context(), img)

	require.NoError(t, err)
	assert.Equal(t, finder.scores, result)
}

func TestTop5MatchesPropagatesFinderError(t *testing.T) {
	img := openImage(t, "cardImage.jpg")
	expected := assert.AnError
	finder := &recordingFinder{err: expected}

	_, err := detection.NewMatcher(finder, detection.NewHasher()).Top5Matches(t.Context(), img)

	require.ErrorIs(t, err, expected)
}

func TestTop5MatchesFailsBeforeLookupOnBadImage(t *testing.T) {
	img := bytes.NewReader([]byte("no image"))
	finder := &recordingFinder{}

	_, err := detection.NewMatcher(finder, detection.NewHasher()).Top5Matches(t.Context(), img)

	require.Error(t, err)
	assert.False(t, finder.called)
}
