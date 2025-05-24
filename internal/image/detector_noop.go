//go:build !opencv

package image

import "io"

type noopDetector struct {
}

func (d noopDetector) Detect(in io.Reader) (Images, error) {
	return NewImages(), nil
}

func NewDetector() Detector {
	return noopDetector{}
}
