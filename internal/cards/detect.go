package cards

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
)

type Score struct {
	ID    ID
	Score int
}

type Scores []Score

func (s Scores) Find(oID ID) *Score {
	for _, score := range s {
		if score.ID.Eq(oID) {
			return &score
		}
	}

	return nil
}

type Match struct {
	Card
	Confidence int
}

type Matches struct {
	PagedResult[Match]
}

func NewMatches(cards Cards, scores Scores, page Page) Matches {
	matches := make([]Match, 0)
	for _, c := range cards.Result {
		if s := scores.Find(c.ID); s != nil {
			matches = append(matches, Match{Card: c, Confidence: s.Score})
		}
	}

	// sort by lowest-distance, lowest = best match
	slices.SortStableFunc(matches, func(a, b Match) int {
		return cmp.Compare(a.Confidence, b.Confidence)
	})

	return Matches{
		NewPagedResult(matches, page),
	}
}

func EmptyMatches(p Page) Matches {
	return Matches{
		NewEmptyResult[Match](p),
	}
}

// PHashThreshold is the max Hamming distance for a 256-bit pHash to count as a match (~23%).
const PHashThreshold = 60

// DHashThreshold is the max Hamming distance for a dhash to count as a match. Looser
// than PHashThreshold's ratio since dhash is noisier for camera-vs-scan comparisons.
const DHashThreshold = 28

type Hash struct {
	PHashR []uint64
	PHashG []uint64
	PHashB []uint64
	DHash  uint64
}

// asBase2 concatenates the given 64-bit words into a single base-2 string.
func asBase2(words []uint64) string {
	var sb strings.Builder
	for _, v := range words {
		fmt.Fprintf(&sb, "%064b", v)
	}

	return sb.String()
}

func (h Hash) PHashRBase2() string {
	return asBase2(h.PHashR)
}

func (h Hash) PHashGBase2() string {
	return asBase2(h.PHashG)
}

func (h Hash) PHashBBase2() string {
	return asBase2(h.PHashB)
}

func (h Hash) DHashBase2() string {
	return fmt.Sprintf("%064b", h.DHash)
}

type DetectRepository interface {
	Top5MatchesByHash(ctx context.Context, hashes ...Hash) (Scores, error)
}

type Detector interface {
	Detect(img io.Reader) ([]Detectable, error)
}

type DetectService struct {
	cRepo    CardRepository
	dRepo    DetectRepository
	detector Detector
}

func NewDetectService(cRepo CardRepository, dRepo DetectRepository, detector Detector) *DetectService {
	return &DetectService{
		cRepo:    cRepo,
		dRepo:    dRepo,
		detector: detector,
	}
}

type Degree int

// Degree rotation angle in degrees.
const (
	None      Degree = 0
	Degree90  Degree = 90
	Degree180 Degree = 180
)

type Detectable interface {
	Rotate(angle Degree) Detectable
	Hash() (Hash, error)
}

func (s *DetectService) Detect(ctx context.Context, c Collector, in io.Reader) (Matches, error) {
	result, dErr := s.detector.Detect(in)
	if dErr != nil {
		return Matches{}, aerrors.NewUnknownError(dErr, "detection-failed")
	}

	hashes := make([]Hash, 0)
	for _, r := range result {
		hash, err := r.Hash()
		if err != nil {
			return Matches{}, aerrors.NewUnknownError(err, "hashing-failed")
		}
		hashes = append(hashes, hash)

		rhash, err := r.Rotate(Degree180).Hash()
		if err != nil {
			return Matches{}, aerrors.NewUnknownError(err, "rotated-hashing-failed")
		}
		hashes = append(hashes, rhash)
	}

	return s.DetectByHash(ctx, c, hashes...)
}

func (s *DetectService) DetectByHash(ctx context.Context, c Collector, hashes ...Hash) (Matches, error) {
	if len(hashes) == 0 {
		return EmptyMatches(DefaultPage()), nil
	}

	scores, err := s.dRepo.Top5MatchesByHash(ctx, hashes...)
	if err != nil {
		return Matches{}, aerrors.NewUnknownError(err, "unable-to-execute-hash-search")
	}

	if len(scores) == 0 {
		return EmptyMatches(DefaultPage()), nil
	}

	filter := NewFilter().WithCollector(c)
	for _, s := range scores {
		filter = filter.WithID(s.ID)
	}
	limit := 5
	page := NewPage(1, limit)
	cards, err := s.cRepo.Find(ctx, filter, page)
	if err != nil {
		return Matches{}, aerrors.NewUnknownError(err, "unable-to-execute-card-search-by-id")
	}

	matches := NewMatches(cards, scores, page)
	matches.HasMore = false

	return matches, nil
}
