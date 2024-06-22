package detect

import (
	"context"
	"io"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
)

type Service interface {
	Detect(ctx context.Context, collector cards.Collector, in io.Reader) (Matches, error)
}

type detectService struct {
	repo     Repository
	detector Detector
	hasher   Hasher
}

func NewDetectService(repo Repository, detector Detector, hasher Hasher) Service {
	return &detectService{
		repo:     repo,
		detector: detector,
		hasher:   hasher,
	}
}

func (s *detectService) Detect(ctx context.Context, collector cards.Collector, in io.Reader) (Matches, error) {
	result, err := s.detector.Detect(in)
	if err != nil {
		return nil, aerrors.NewUnknownError(err, "detection-failed")
	}

	hashes := make([]Hash, 0)
	for _, r := range result {
		hash, err := s.hasher.Hash(r)
		if err != nil {
			return nil, aerrors.NewUnknownError(err, "hashing-failed")
		}
		hashes = append(hashes, hash)

		rhash, err := s.hasher.Hash(r.Rotate(Degree180))
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
