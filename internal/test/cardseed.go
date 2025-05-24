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

func CardSeed() ([]cards.Card, error) {
	_, cf, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get current dir")
	}
	dir := path.Join(path.Dir(cf))

	files := []string{
		"testdata/search.json",
		"testdata/collected.json",
		"testdata/detect.json",
		"testdata/more.json",
		"testdata/detail.json",
	}
	seed := make([]cards.Card, 0)
	for _, f := range files {
		cc, err := readFromJSON(filepath.Join(dir, f))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s, %w", f, err)
		}
		seed = append(seed, cc...)
	}

	return seed, nil
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
