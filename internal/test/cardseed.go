package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

func currentDir() string {
	_, cf, _, _ := runtime.Caller(0)

	return path.Join(path.Dir(cf))
}

func CardSeed() ([]cards.Card, error) {
	dir := currentDir()

	cc := make([]cards.Card, 0)

	cards10E, err := readFromJSON(filepath.Join(dir, "testdata/cards10E.json"))
	if err != nil {
		return nil, err
	}
	cc = append(cc, cards10E...)

	cards2ED, err := readFromJSON(filepath.Join(dir, "testdata/cards2ED.json"))
	if err != nil {
		return nil, err
	}
	cc = append(cc, cards2ED...)

	cards2X2, err := readFromJSON(filepath.Join(dir, "testdata/cards2X2.json"))
	if err != nil {
		return nil, err
	}
	cc = append(cc, cards2X2...)

	return cc, nil
}

func readFromJSON(path string) ([]cards.Card, error) {
	// #nosec G304 only used in tests
	cardsRaw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s, %w", path, err)
	}

	var cards []cards.Card
	if err := json.Unmarshal(cardsRaw, &cards); err != nil {
		return nil, err
	}

	return cards, nil
}
