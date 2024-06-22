package fakes

import (
	"image"
	"io"

	_ "image/jpeg"

	"github.com/konstantinfoerster/card-service-go/internal/cards/detect"
)

func NewDetector() detect.Detector {
	return fakeDetector{}
}

type fakeDetector struct {
}

func (d fakeDetector) Detect(in io.Reader) (detect.Images, error) {
	dImg, _, err := image.Decode(in)
	if err != nil {
		return nil, err
	}

	return detect.Images{detect.Image{Image: dImg}}, nil
}
