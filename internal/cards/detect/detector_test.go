package detect_test

import (
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/cards/detect"
	"github.com/stretchr/testify/assert"
)

func TestNewImages(t *testing.T) {
	actual := detect.NewImages()

	assert.NotNil(t, actual)
}
