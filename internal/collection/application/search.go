package application

import (
	"context"
	"io"

	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/img"
)

type SearchService interface {
	Search(ctx context.Context, collector domain.Collector, name string, page common.Page) (domain.Cards, error)
	Detect(ctx context.Context, collector domain.Collector, in io.Reader) (domain.Matches, error)
}

type searchService struct {
	repo     domain.SearchRepository
	detector img.Detector
	hasher   img.Hasher
}

func NewSearchService(repo domain.SearchRepository, detector img.Detector, hasher img.Hasher) SearchService {
	return &searchService{
		repo:     repo,
		detector: detector,
		hasher:   hasher,
	}
}

func (s *searchService) Search(_ context.Context, collector domain.Collector,
	name string, page common.Page) (domain.Cards, error) {
	if collector.ID == "" {
		r, err := s.repo.FindByName(name, page)
		if err != nil {
			return domain.Empty(page), common.NewUnknownError(err, "unable-to-execute-search")
		}

		return r, nil
	}

	r, err := s.repo.FindByCollectorAndName(collector, name, page)
	if err != nil {
		return domain.Empty(page), common.NewUnknownError(err, "unable-to-execute-search")
	}

	return r, nil
}

func (s *searchService) Detect(ctx context.Context, collector domain.Collector, in io.Reader) (domain.Matches, error) {
	result, err := s.detector.Detect(in)
	if err != nil {
		return nil, common.NewUnknownError(err, "detection-failed")
	}

	hashes := make([]domain.Hash, 0)
	for _, r := range result {
		hash, err := s.hasher.Hash(r)
		if err != nil {
			return nil, common.NewUnknownError(err, "hashing-failed")
		}
		hashes = append(hashes, hash)

		rhash, err := s.hasher.Hash(r.Rotate(img.Degree180))
		if err != nil {
			return nil, common.NewUnknownError(err, "rotated-hashing-failed")
		}
		hashes = append(hashes, rhash)
	}

	if collector.ID == "" {
		r, err := s.repo.Top5MatchesByHash(ctx, hashes...)
		if err != nil {
			return nil, common.NewUnknownError(err, "unable-to-execute-hash-search")
		}

		return r, nil
	}

	r, err := s.repo.Top5MatchesByCollectorAndHash(ctx, collector, hashes...)
	if err != nil {
		return nil, common.NewUnknownError(err, "unable-to-execute-hash-search")
	}

	return r, nil
}
