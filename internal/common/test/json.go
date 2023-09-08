package commontest

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func ToJSON(t *testing.T, v interface{}) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	require.NoError(t, err)

	return b
}

func FromJSON[T any](t *testing.T, resp *http.Response) *T {
	t.Helper()

	v := new(T)
	err := json.NewDecoder(resp.Body).Decode(v)
	require.NoError(t, err)

	return v
}
