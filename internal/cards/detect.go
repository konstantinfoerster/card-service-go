package cards

import (
	"context"
	"io"

	"github.com/gofiber/fiber/v2/log"
	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/common/detect"
)

type Match struct {
	Card
	Score int
}

type Matches []Match

type DetectRepository interface {
	Top5MatchesByHash(ctx context.Context, hashes ...detect.Hash) (Matches, error)
	Top5MatchesByCollectorAndHash(ctx context.Context, collector Collector, hashes ...detect.Hash) (Matches, error)
}

type DetectService interface {
	Detect(ctx context.Context, collector Collector, in io.Reader) (Matches, error)
}

type detectService struct {
	repo     DetectRepository
	detector detect.Detector
	hasher   detect.Hasher
}

func NewDetectService(repo DetectRepository, detector detect.Detector, hasher detect.Hasher) DetectService {
	return &detectService{
		repo:     repo,
		detector: detector,
		hasher:   hasher,
	}
}

func (s *detectService) Detect(ctx context.Context, collector Collector, in io.Reader) (Matches, error) {
	log.Info("x 1")
	result, err := s.detector.Detect(in)
	if err != nil {
		return nil, aerrors.NewUnknownError(err, "detection-failed")
	}
	log.Info("x 2")

	hashes := make([]detect.Hash, 0)
	for _, r := range result {
		hash, err := s.hasher.Hash(r)
		if err != nil {
			return nil, aerrors.NewUnknownError(err, "hashing-failed")
		}
		log.Info("x 3")
		hashes = append(hashes, hash)

		rhash, err := s.hasher.Hash(r.Rotate(detect.Degree180))
		if err != nil {
			return nil, aerrors.NewUnknownError(err, "rotated-hashing-failed")
		}
		log.Info("x 4")
		hashes = append(hashes, rhash)
	}

	if collector.ID == "" {
		r, err := s.repo.Top5MatchesByHash(ctx, hashes...)
		if err != nil {
			return nil, aerrors.NewUnknownError(err, "unable-to-execute-hash-search")
		}
		log.Info("x 5")

		return r, nil
	}

	log.Info("x 6")
	r, err := s.repo.Top5MatchesByCollectorAndHash(ctx, collector, hashes...)
	if err != nil {
		return nil, aerrors.NewUnknownError(err, "unable-to-execute-hash-search")
	}
	log.Info("x 7")

	return r, nil
}
