package test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func Base64Encoded(t *testing.T, token any) string {
	rawJwToken, err := json.Marshal(token)
	require.NoError(t, err)

	return base64.URLEncoding.EncodeToString(rawJwToken)
}
