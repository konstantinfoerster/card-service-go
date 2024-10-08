package image

import (
	"io"
)

func NewFakeDetector() Detector {
	return fakeDetector{}
}

type fakeDetector struct {
}

func (d fakeDetector) Detect(in io.Reader) (Images, error) {
	if in == nil {
		return nil, ErrInvalidInput
	}

	img, err := NewImage(in)
	if err != nil {
		return nil, err
	}

	return Images{img}, nil
}
