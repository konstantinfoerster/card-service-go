package imaging

import (
	"errors"
	"fmt"
	"image"
	"image/color"
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

	red, green, blue := splitChannels(img)

	rPHash, err := goimagehash.ExtPerceptionHash(red, width, height)
	if err != nil {
		return cards.Hash{}, fmt.Errorf("failed create red channel phash %w", err)
	}

	gPHash, err := goimagehash.ExtPerceptionHash(green, width, height)
	if err != nil {
		return cards.Hash{}, fmt.Errorf("failed create green channel phash %w", err)
	}

	bPHash, err := goimagehash.ExtPerceptionHash(blue, width, height)
	if err != nil {
		return cards.Hash{}, fmt.Errorf("failed create blue channel phash %w", err)
	}

	dHash, err := goimagehash.DifferenceHash(img)
	if err != nil {
		return cards.Hash{}, fmt.Errorf("failed create dhash %w", err)
	}

	return cards.Hash{
		PHashR: rPHash.GetHash(),
		PHashG: gPHash.GetHash(),
		PHashB: bPHash.GetHash(),
		DHash:  dHash.GetHash(),
	}, nil
}

// splitChannels extracts the R, G and B channels of given img into three
// separate *image.Gray images.
//
// image.Gray stores one value per pixel and satisfies the image.Image type.
func splitChannels(src image.Image) (*image.Gray, *image.Gray, *image.Gray) {
	bounds := src.Bounds()
	red := image.NewGray(bounds)
	green := image.NewGray(bounds)
	blue := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := src.At(x, y).RGBA()
			red.SetGray(x, y, color.Gray{Y: to8Bit(r)})
			green.SetGray(x, y, color.Gray{Y: to8Bit(g)})
			blue.SetGray(x, y, color.Gray{Y: to8Bit(b)})
		}
	}

	return red, green, blue
}

// to8Bit converts a 16-bit value into an 8-bit value.
// Values above 255 are capped at 255.
func to8Bit(v uint32) uint8 {
	v /= 256
	if v > 255 {
		v = 255
	}

	return uint8(v)
}
