package fakes

import (
	"image"
	"io"

	_ "image/jpeg"

	"github.com/konstantinfoerster/card-service-go/internal/common/img"
)

func NewDetector() img.Detector {
	return fakeDetector{}
}

type fakeDetector struct {
}

func (d fakeDetector) Detect(in io.Reader) (img.Images, error) {
	dImg, _, err := image.Decode(in)
	if err != nil {
		return nil, err
	}

	return img.Images{img.Image{Image: dImg}}, nil
}
