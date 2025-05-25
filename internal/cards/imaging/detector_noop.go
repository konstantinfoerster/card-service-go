//go:build !opencv

package imaging

import (
	"io"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type NoopDetector struct {
}

func NewDetector() *NoopDetector {
	return &NoopDetector{}
}

func (d NoopDetector) Detect(in io.Reader) ([]cards.Detectable, error) {
	return make([]cards.Detectable, 0), nil
}
