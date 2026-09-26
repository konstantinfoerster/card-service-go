package detection_test

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashesAllowsNilReader(t *testing.T) {
	_, err := detection.NewHasher().Hashes(nil, detection.None)

	require.ErrorIs(t, err, cards.ErrInvalidInput)
}

func TestHashesFailsOnImageAbovePixelLimit(t *testing.T) {
	const size = 1001 // 1001x1001 is just above detection.MaxPixels
	require.Greater(t, size*size, detection.MaxPixels)

	var img bytes.Buffer
	require.NoError(t, jpeg.Encode(&img, image.NewGray(image.Rect(0, 0, size, size)), nil))

	_, err := detection.NewHasher().Hashes(&img, detection.None)

	require.ErrorIs(t, err, detection.ErrImageTooLarge)
	require.ErrorIs(t, err, cards.ErrInvalidInput)
}

func TestHashesFailsOnNonImage(t *testing.T) {
	img := bytes.NewReader([]byte("no image"))

	_, err := detection.NewHasher().Hashes(img, detection.None)

	require.ErrorIs(t, err, image.ErrFormat)
}

func TestHashesReturnsOneHashPerAngle(t *testing.T) {
	cases := []struct {
		name   string
		angles []detection.Degree
		want   int
	}{
		{
			name:   "no angles",
			angles: nil,
			want:   0,
		},
		{
			name:   "upright only",
			angles: []detection.Degree{detection.None},
			want:   1,
		},
		{
			name:   "upright and upside down",
			angles: []detection.Degree{detection.None, detection.Degree180},
			want:   2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			img := openImage(t, "cardImage.jpg")

			hashes, err := detection.NewHasher().Hashes(img, tc.angles...)

			require.NoError(t, err)
			assert.Len(t, hashes, tc.want)
		})
	}
}

func TestHashesIsDeterministic(t *testing.T) {
	hasher := detection.NewHasher()

	first, err := hasher.Hashes(openImage(t, "cardImage.jpg"), detection.None)
	require.NoError(t, err)
	second, err := hasher.Hashes(openImage(t, "cardImage.jpg"), detection.None)
	require.NoError(t, err)

	assert.Equal(t, first, second)
}

func TestHashValuesAreStable(t *testing.T) {
	img := openImage(t, "cardImage.jpg")

	hashes, err := detection.NewHasher().Hashes(img, detection.None)
	require.NoError(t, err)

	channel := []uint64{
		0x81ff01ff00ff00ff,
		0x00ff00ff00ff00ff,
		0x00ff00ff00ff00ff,
		0x00ff00ff007f003f,
	}
	assert.Equal(t, detection.Hash{
		PHashR: channel,
		PHashG: channel,
		PHashB: channel,
		DHash:  0xc000000000000000,
	}, hashes[0])
}

func openImage(t *testing.T, name string) io.Reader {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)

	return bytes.NewReader(raw)
}
