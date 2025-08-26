package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
)

var ErrGenerate = errors.New("value generation failed")

type RandomGenerator struct {
}

func NewRandomGenerator() RandomGenerator {
	return RandomGenerator{}
}

func (g RandomGenerator) Generate() ([]byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, errors.Join(err, ErrGenerate)
	}

	return b, nil
}

type StaticGenerator struct {
	Value string
}

func (g StaticGenerator) Generate() ([]byte, error) {
	return []byte(g.Value), nil
}

func (g StaticGenerator) MustGenerateBase64Encoded() string {
	val, err := g.Generate()
	if err != nil {
		panic(err)
	}

	return base64.RawURLEncoding.EncodeToString(val)
}
