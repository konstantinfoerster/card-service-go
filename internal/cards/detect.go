package cards

import (
	"context"
	"errors"
	"io"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/image"
)

var (
	ErrMatchNotFound = errors.New("no match found")
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

func NewMatches(cards Cards, scores Scores, page Page) Matches {
	matches := make([]Match, 0)
	for _, c := range cards.Result {
		if s := scores.Find(c.ID); s != nil {
			matches = append(matches, Match{Card: c, Confidence: s.Score})
		}
	}

	return Matches{
		NewPagedResult(matches, page),
	}
}

type Matches struct {
	PagedResult[Match]
}

func EmptyMatches(p Page) Matches {
	return Matches{
		NewEmptyResult[Match](p),
	}
}

type DetectRepository interface {
	Top5MatchesByHash(ctx context.Context, hashes ...image.Hash) (Scores, error)
}

type DetectService interface {
	Detect(ctx context.Context, collector Collector, in io.Reader) (Matches, error)
}

type detectService struct {
	cRepo    CardRepository
	dRepo    DetectRepository
	detector image.Detector
	hasher   image.Hasher
}

func NewDetectService(cRepo CardRepository, dRepo DetectRepository, detector image.Detector,
	hasher image.Hasher) DetectService {
	return &detectService{
		cRepo:    cRepo,
		dRepo:    dRepo,
		detector: detector,
		hasher:   hasher,
	}
}

func (s *detectService) Detect(ctx context.Context, c Collector, in io.Reader) (Matches, error) {
	result, err := s.detector.Detect(in)
	if err != nil {
		return EmptyMatches(DefaultPage()), aerrors.NewUnknownError(err, "detection-failed")
	}

	hashes := make([]image.Hash, 0)
	for _, r := range result {
		hash, err := s.hasher.Hash(r)
		if err != nil {
			return EmptyMatches(DefaultPage()), aerrors.NewUnknownError(err, "hashing-failed")
		}
		hashes = append(hashes, hash)

		rhash, err := s.hasher.Hash(r.Rotate(image.Degree180))
		if err != nil {
			return EmptyMatches(DefaultPage()), aerrors.NewUnknownError(err, "rotated-hashing-failed")
		}
		hashes = append(hashes, rhash)
	}

	scores, err := s.dRepo.Top5MatchesByHash(ctx, hashes...)
	if err != nil {
		return EmptyMatches(DefaultPage()), aerrors.NewUnknownError(err, "unable-to-execute-hash-search")
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
		return EmptyMatches(DefaultPage()), aerrors.NewUnknownError(err, "unable-to-execute-card-search-by-id")
	}

	matches := NewMatches(cards, scores, page)
	matches.HasMore = false

	return matches, nil
}
