package image_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/image"
	"github.com/stretchr/testify/assert"
)

func TestNewImages(t *testing.T) {
	actual := image.NewImages()

	assert.NotNil(t, actual)
}
