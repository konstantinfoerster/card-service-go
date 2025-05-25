package imaging

import (
	"image"
	"io"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type FakeDetector struct {
}

func NewFakeDetector() *FakeDetector {
	return &FakeDetector{}
}

func (d FakeDetector) Detect(in io.Reader) ([]cards.Detectable, error) {
	if in == nil {
		return nil, ErrInvalidInput
	}

	dImg, _, err := image.Decode(in)
	if err != nil {
		return nil, err
	}

	return []cards.Detectable{Image{dImg}}, nil
}
