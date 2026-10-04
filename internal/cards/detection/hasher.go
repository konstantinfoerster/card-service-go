package detection

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"

	_ "image/jpeg"
	_ "image/png"

	"github.com/anthonynsimon/bild/transform"
	"github.com/corona10/goimagehash"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

var ErrMissingInput = errors.New("missing input")

var ErrImageTooLarge = errors.New("image too large")

// MaxPixels is the maximum size of a decoded image, it bounds the memory a single
// request can allocate.
const MaxPixels = 1000 * 1000

type Hash struct {
	PHashR []uint64
	PHashG []uint64
	PHashB []uint64
	DHash  uint64
}

type Degree int

// Degree rotation angle in degrees.
const (
	None      Degree = 0
	Degree180 Degree = 180
)

type Hasher struct {
}

func NewHasher() Hasher {
	return Hasher{}
}

func (h Hasher) Hashes(in io.Reader, angles ...Degree) ([]Hash, error) {
	if in == nil {
		return nil, errors.Join(ErrMissingInput, cards.ErrInvalidInput)
	}

	// buffer the header so it can be read again after
	// DecodeConfig consumed it
	var head bytes.Buffer
	cfg, _, err := image.DecodeConfig(io.TeeReader(in, &head))
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to read image config %w", err), cards.ErrInvalidInput)
	}

	if pixels := cfg.Width * cfg.Height; pixels > MaxPixels {
		return nil, errors.Join(
			fmt.Errorf("image %dx%d has %d pixels, allowed are %d: %w",
				cfg.Width, cfg.Height, pixels, MaxPixels, ErrImageTooLarge),
			cards.ErrInvalidInput)
	}

	r := io.MultiReader(bytes.NewReader(head.Bytes()), in)
	dImg, _, err := image.Decode(r)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to decode image %w", err), cards.ErrInvalidInput)
	}

	hashes := make([]Hash, 0, len(angles))
	for _, a := range angles {
		hash, err := Image{dImg}.Rotate(a).Hash()
		if err != nil {
			return nil, fmt.Errorf("failed to hash image rotated by %d degrees, %w", a, err)
		}

		hashes = append(hashes, hash)
	}

	return hashes, nil
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

func (img Image) Hash() (Hash, error) {
	width := 16
	height := 16

	red, green, blue := splitChannels(img)

	rPHash, err := goimagehash.ExtPerceptionHash(red, width, height)
	if err != nil {
		return Hash{}, fmt.Errorf("failed create red channel phash %w", err)
	}

	gPHash, err := goimagehash.ExtPerceptionHash(green, width, height)
	if err != nil {
		return Hash{}, fmt.Errorf("failed create green channel phash %w", err)
	}

	bPHash, err := goimagehash.ExtPerceptionHash(blue, width, height)
	if err != nil {
		return Hash{}, fmt.Errorf("failed create blue channel phash %w", err)
	}

	dHash, err := goimagehash.DifferenceHash(img)
	if err != nil {
		return Hash{}, fmt.Errorf("failed create dhash %w", err)
	}

	return Hash{
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
