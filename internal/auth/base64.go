package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

var (
	errValueRequired       = errors.New("value must not be empty")
	errInvalidEncoding     = errors.New("invalid value encoding")
	errInvalidValueContent = errors.New("invalid value content")
	errInvalidValue        = errors.New("invalid value")
)

func decodeBase64[T any](value string) (*T, error) {
	if strings.TrimSpace(value) == "" {
		return nil, errValueRequired
	}

	rawJSON, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return nil, errInvalidEncoding
	}

	target := new(T)
	if err = json.Unmarshal(rawJSON, target); err != nil {
		return nil, errors.Join(err, errInvalidValueContent)
	}

	return target, nil
}

func encodeBase64[T any](value T) (string, error) {
	rawJwToken, err := json.Marshal(&value)
	if err != nil {
		return "", errInvalidValue
	}

	return base64.URLEncoding.EncodeToString(rawJwToken), nil
}
