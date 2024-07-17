package detect

import (
	"fmt"
	"image"
	"io"

	_ "image/jpeg"
)

func NewFakeDetector() Detector {
	return fakeDetector{}
}

type fakeDetector struct {
}

func (d fakeDetector) Detect(in io.Reader) (Images, error) {
    if in == nil {
        return nil, fmt.Errorf("image require for detection")
    }

	dImg, _, err := image.Decode(in)
	if err != nil {
		return nil, err
	}

	return Images{Image{Image: dImg}}, nil
}
