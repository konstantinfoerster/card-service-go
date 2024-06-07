package img

import (
	"image"

	"github.com/anthonynsimon/bild/transform"
)

type Degree int

const (
	None Degree = iota
	Degree90
	Degree180
)

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
