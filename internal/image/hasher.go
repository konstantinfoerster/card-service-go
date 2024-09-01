package image

import (
	"fmt"

	"github.com/corona10/goimagehash"
)

type Hash struct {
	Value []uint64
	Bits  int
}

func (h Hash) AsBase2() []string {
	base2 := make([]string, 0, len(h.Value))
	for _, v := range h.Value {
		base2 = append(base2, fmt.Sprintf("%064b", v))
	}

	return base2
}

type Hasher interface {
	// Hash builds a hash based on the given image
	Hash(img Image) (Hash, error)
	// Distance builds the distance between the given hashes
	Distance(hash1 Hash, hash2 Hash) (int, error)
}

func NewPHasher() Hasher {
	return phasher{}
}

type phasher struct {
}

func (p phasher) Hash(img Image) (Hash, error) {
	width := 16
	height := 16
	ph, err := goimagehash.ExtPerceptionHash(img, width, height)
	if err != nil {
		return Hash{}, fmt.Errorf("failed create phash %w", err)
	}

	return Hash{
		Value: ph.GetHash(),
		Bits:  ph.Bits(),
	}, nil
}

func (p phasher) Distance(h1 Hash, h2 Hash) (int, error) {
	ph1 := goimagehash.NewExtImageHash(h1.Value, goimagehash.PHash, h1.Bits)
	ph2 := goimagehash.NewExtImageHash(h2.Value, goimagehash.PHash, h2.Bits)

	return ph1.Distance(ph2)
}
