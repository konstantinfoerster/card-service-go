package imaging

import (
	"errors"
	"fmt"
	"image"
	"io"

	_ "image/jpeg"

	"github.com/anthonynsimon/bild/transform"
	"github.com/corona10/goimagehash"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

var ErrInvalidInput = errors.New("invalid input reader")

type Image struct {
	image.Image
}

func NewImage(in io.Reader) (Image, error) {
	if in == nil {
		return Image{}, ErrInvalidInput
	}

	dImg, _, err := image.Decode(in)
	if err != nil {
		return Image{}, err
	}

	return Image{dImg}, nil
}

func (img Image) Rotate(angle cards.Degree) cards.Detectable {
	if angle == cards.None {
		return img
	}

	rImg := transform.Rotate(img, float64(angle), nil)

	return Image{rImg}
}

func (img Image) Hash() (cards.Hash, error) {
	width := 16
	height := 16
	ph, err := goimagehash.ExtPerceptionHash(img, width, height)
	if err != nil {
		return cards.Hash{}, fmt.Errorf("failed create phash %w", err)
	}

	return cards.Hash{
		Value: ph.GetHash(),
		Bits:  ph.Bits(),
	}, nil
}

func Distance(h1 cards.Hash, h2 cards.Hash) (int, error) {
	ph1 := goimagehash.NewExtImageHash(h1.Value, goimagehash.PHash, h1.Bits)
	ph2 := goimagehash.NewExtImageHash(h2.Value, goimagehash.PHash, h2.Bits)

	return ph1.Distance(ph2)
}
