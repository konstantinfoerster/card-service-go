package img_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/common/img"
	"github.com/stretchr/testify/assert"
)

func TestNewImages(t *testing.T) {
	actual := img.NewImages()

	assert.NotNil(t, actual)
}
