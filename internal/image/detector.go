package image

import (
	"image"
	"io"

	_ "image/jpeg"

	"github.com/anthonynsimon/bild/transform"
)

type Degree int

const (
	None Degree = iota
	Degree90
	Degree180
)

func NewImage(in io.Reader) (Image, error) {
	dImg, _, err := image.Decode(in)
	if err != nil {
		return Image{}, err
	}

	return Image{dImg}, nil
}

type Image struct {
	image.Image
}

func (img Image) Rotate(angle Degree) Image {
	if angle == None {
		return img
	}

	rImg := transform.Rotate(img, float64(angle), nil)

	return Image{rImg}
}

type Images []Image

func NewImages() Images {
	return make(Images, 0)
}

type Detector interface {
	Detect(img io.Reader) (Images, error)
}
