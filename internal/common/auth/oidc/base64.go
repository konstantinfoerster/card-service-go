package oidc

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
)

func DecodeBase64[T any](value string) (*T, error) {
	if strings.TrimSpace(value) == "" {
		return nil, aerrors.NewInvalidInputMsg("unable-to-decode-value", "empty value")
	}

	rawJSON, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return nil, aerrors.NewInvalidInputError(err, "unable-to-decode-value", "invalid encoding")
	}

	target := new(T)
	if err = json.Unmarshal(rawJSON, target); err != nil {
		return nil, err
	}

	return target, nil
}

func EncodeBase64[T any](value T) (string, error) {
	rawJwToken, err := json.Marshal(&value)
	if err != nil {
		return "", aerrors.NewUnknownError(err, "unable-to-encode-value")
	}

	return base64.URLEncoding.EncodeToString(rawJwToken), nil
}
