package img

import (
	"io"
)

type Detector interface {
	Detect(img io.Reader) (Images, error)
}
