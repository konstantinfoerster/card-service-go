package test

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func ToJSON(t *testing.T, v any) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	require.NoError(t, err)

	return b
}

func FromJSON[T any](t *testing.T, r io.Reader) *T {
	t.Helper()

	v := new(T)
	err := json.NewDecoder(r).Decode(v)
	require.NoError(t, err)

	return v
}

func ToString(t *testing.T, r io.Reader) string {
	t.Helper()

	body, err := io.ReadAll(r)
	require.NoError(t, err)

	return string(body)
}

func ToBytes(t *testing.T, r io.Reader) []byte {
	t.Helper()

	body, err := io.ReadAll(r)
	require.NoError(t, err)

	return body
}
