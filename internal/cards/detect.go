package cards

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"

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

	return Matches{
		NewPagedResult(matches, page),
	}
}

func EmptyMatches(p Page) Matches {
	return Matches{
		NewEmptyResult[Match](p),
	}
}

type Hash struct {
	Value []uint64
	Bits  int
}

func FromHexString(hash string) (Hash, error) {
	if len(hash) != 64 {
		return Hash{}, fmt.Errorf("invalid hash length: expected 64 characters, got %d", len(hash))
	}

	b, err := hex.DecodeString(hash)
	if err != nil || len(b) != 32 {
		return Hash{}, fmt.Errorf("invalid hash bytes")
	}

	// 2. Jeweils 8 Bytes direkt als uint64 interpretieren (BigEndian)
	v := []uint64{
		binary.BigEndian.Uint64(b[0:8]),
		binary.BigEndian.Uint64(b[8:16]),
		binary.BigEndian.Uint64(b[16:24]),
		binary.BigEndian.Uint64(b[24:32]),
	}
	return Hash{Value: v, Bits: 256}, nil
}

func (h Hash) AsBase2() []string {
	base2 := make([]string, 0, len(h.Value))
	for _, v := range h.Value {
		base2 = append(base2, fmt.Sprintf("%064b", v))
	}

	return base2
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

const (
	None Degree = iota
	Degree90
	Degree180
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

	for _, s := range scores {
		slog.Info("detected ", slog.Int("id", s.ID.CardID))
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
