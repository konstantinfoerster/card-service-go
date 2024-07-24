package cards

import (
	"context"
	"io"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/image"
)

type Match struct {
	Card
	Score int
}

type Matches []Match

type DetectRepository interface {
	Top5MatchesByHash(ctx context.Context, hashes ...image.Hash) (Matches, error)
	Top5MatchesByCollectorAndHash(ctx context.Context, collector Collector, hashes ...image.Hash) (Matches, error)
}

type DetectService interface {
	Detect(ctx context.Context, collector Collector, in io.Reader) (Matches, error)
}

type detectService struct {
	repo     DetectRepository
	detector image.Detector
	hasher   image.Hasher
}

func NewDetectService(repo DetectRepository, detector image.Detector, hasher image.Hasher) DetectService {
	return &detectService{
		repo:     repo,
		detector: detector,
		hasher:   hasher,
	}
}

func (s *detectService) Detect(ctx context.Context, collector Collector, in io.Reader) (Matches, error) {
	result, err := s.detector.Detect(in)
	if err != nil {
		return nil, aerrors.NewUnknownError(err, "detection-failed")
	}

	hashes := make([]image.Hash, 0)
	for _, r := range result {
		hash, err := s.hasher.Hash(r)
		if err != nil {
			return nil, aerrors.NewUnknownError(err, "hashing-failed")
		}
		hashes = append(hashes, hash)

		rhash, err := s.hasher.Hash(r.Rotate(image.Degree180))
		if err != nil {
			return nil, aerrors.NewUnknownError(err, "rotated-hashing-failed")
		}
		hashes = append(hashes, rhash)
	}

	if collector.ID == "" {
		r, err := s.repo.Top5MatchesByHash(ctx, hashes...)
		if err != nil {
			return nil, aerrors.NewUnknownError(err, "unable-to-execute-hash-search")
		}

		return r, nil
	}

	r, err := s.repo.Top5MatchesByCollectorAndHash(ctx, collector, hashes...)
	if err != nil {
		return nil, aerrors.NewUnknownError(err, "unable-to-execute-hash-search")
	}

	return r, nil
}
