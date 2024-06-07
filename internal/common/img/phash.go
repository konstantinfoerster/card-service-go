package img

import (
	"fmt"

	"github.com/corona10/goimagehash"
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
)

type Hasher interface {
	// Hash builds a hash based on the given image
	Hash(img Image) (domain.Hash, error)
	// Distance builds the distance between the given hashes
	Distance(hash1 domain.Hash, hash2 domain.Hash) (int, error)
}

func NewPHasher() Hasher {
	return phasher{}
}

type phasher struct {
}

func (p phasher) Hash(img Image) (domain.Hash, error) {
	width := 16
	height := 16
	ph, err := goimagehash.ExtPerceptionHash(img, width, height)
	if err != nil {
		return domain.Hash{}, fmt.Errorf("failed create phash %w", err)
	}

	return domain.Hash{
		Value: ph.GetHash(),
		Bits:  ph.Bits(),
	}, nil
}

func (p phasher) Distance(h1 domain.Hash, h2 domain.Hash) (int, error) {
	ph1 := goimagehash.NewExtImageHash(h1.Value, goimagehash.PHash, h1.Bits)
	ph2 := goimagehash.NewExtImageHash(h2.Value, goimagehash.PHash, h2.Bits)

	return ph1.Distance(ph2)
}
