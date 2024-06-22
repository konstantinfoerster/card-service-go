package detect

import (
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type Match struct {
	cards.Card
	Score int
}

type Matches []Match

type Hash struct {
	Value []uint64
	Bits  int
}

func (h Hash) AsBase2() []string {
	base2 := make([]string, 0, len(h.Value))
	for _, v := range h.Value {
		base2 = append(base2, fmt.Sprintf("%064b", v))
	}

	return base2
}
