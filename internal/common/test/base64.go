package commontest

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/stretchr/testify/require"
)

func Base64Encoded(t *testing.T, token *oidc.JSONWebToken) string {
	rawJwToken, err := json.Marshal(&token)
	require.NoError(t, err)

	return base64.URLEncoding.EncodeToString(rawJwToken)
}
