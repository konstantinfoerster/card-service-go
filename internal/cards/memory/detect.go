package memory

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"time"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detection"
)

// InMemDetectRepository compares a hash with the hashes of all card images.
// The hashes are loaded once, a re-import shows up after a restart.
type InMemDetectRepository struct {
	images []detection.ImageHash
}

func NewDetectRepository(images []detection.ImageHash) *InMemDetectRepository {
	return &InMemDetectRepository{
		images: images,
	}
}

func (r *InMemDetectRepository) Top5MatchesByHash(_ context.Context, hashes ...detection.Hash) (cards.Scores, error) {
	defer cards.TimeTracker(time.Now(), "Top5MatchesByHash")

	// best image per card name, the faces of a split card share one image
	best := make(map[string]cards.Score)
	for _, img := range r.images {
		// the best hash counts, e.g. upright or upside down
		score := detection.MaxScore
		for _, h := range hashes {
			score = min(score, h.Distance(img.Hash))
		}
		if score >= detection.MaxScore {
			continue
		}

		s := cards.Score{ID: img.ID, Score: score}
		if b, ok := best[img.Name]; !ok || compareScores(s, b) < 0 {
			best[img.Name] = s
		}
	}

	scores := slices.SortedFunc(maps.Values(best), compareScores)
	if len(scores) > 5 {
		scores = scores[:5]
	}

	return scores, nil
}

// compareScores orders by score, then by ID, so equal scores keep a fixed order.
func compareScores(a, b cards.Score) int {
	return cmp.Or(
		cmp.Compare(a.Score, b.Score),
		cmp.Compare(a.ID.CardID, b.ID.CardID),
		cmp.Compare(a.ID.FaceID, b.ID.FaceID),
	)
}
