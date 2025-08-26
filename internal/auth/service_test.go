package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var client = &http.Client{}
var staticGenerator = auth.StaticGenerator{Value: "randVal"}

func TestUnsupportedProvider(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		errType  aerrors.ErrorType
	}{
		{
			name:     "Unknown provider",
			provider: "unknown",
			errType:  aerrors.ErrInvalidInput,
		},
		{
			name:     "empty provider",
			provider: "",
			errType:  aerrors.ErrInvalidInput,
		},
		{
			name:     "space only provider",
			provider: "  ",
			errType:  aerrors.ErrInvalidInput,
		},
	}

	for _, tc := range cases {
		t.Run("Authenticate - "+tc.name, func(t *testing.T) {
			ctx := context.Background()
			svc := auth.New(nil, auth.Providers{})

			_, _, err := svc.Authenticate(ctx, tc.provider, "", "")

			var appErr aerrors.AppError
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, tc.errType, appErr.ErrorType)
		})

		t.Run("AuthURL - "+tc.name, func(t *testing.T) {
			svc := auth.New(nil, auth.Providers{})

			_, _, err := svc.AuthURL(tc.provider)

			var appErr aerrors.AppError
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, tc.errType, appErr.ErrorType)
		})
		t.Run("Revoke - "+tc.name, func(t *testing.T) {
			ctx := context.Background()
			svc := auth.New(nil, auth.Providers{})

			err := svc.Revoke(ctx, &auth.JWT{Provider: tc.provider})

			var appErr aerrors.AppError
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, tc.errType, appErr.ErrorType)
		})
	}
}

func TestAuthURL(t *testing.T) {
	cfg := auth.ProviderCfg{
		AuthURL:     "http://localhost/oauth2/auth",
		RedirectURI: "http://localhost",
		ClientID:    "client id 0",
		Scopes:      []string{"openid", "email"},
	}
	randVal := staticGenerator.MustGenerateBase64Encoded()
	codeChallenge := sha256.Sum256([]byte(randVal))
	expectedURL := "http://localhost/oauth2/auth?" + url.Values{
		"state":                 {randVal},
		"code_challenge_method": {"S256"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(codeChallenge[:])},
		"client_id":             {"client id 0"},
		"redirect_uri":          {"http://localhost"},
		"scope":                 {"openid email"},
		"response_type":         {"code"},
		"access_type":           {"online"},
	}.Encode()
	expectedState := auth.State{
		Value:    randVal,
		Verifier: randVal,
	}
	providers := auth.NewProviders(NewTestProvider(cfg, client))
	svc := auth.New(staticGenerator, providers)

	actualURL, state, err := svc.AuthURL("test")

	require.NoError(t, err)
	assert.Equal(t, expectedState, state)
	assert.Equal(t, expectedURL, actualURL)
}

func TestAuthenticate(t *testing.T) {
	reqAssert := func(uri, body string) {
		expectedBody := url.Values{
			"code":          {"code-0"},
			"code_verifier": {"someVerifier"},
			"client_id":     {"client id 0"},
			"client_secret": {"secure"},
			"redirect_uri":  {"http://localhost"},
			"grant_type":    {"authorization_code"},
		}
		if strings.HasSuffix(uri, "/auth") {
			assert.Equal(t, expectedBody.Encode(), body)
		}
	}
	srv := startProviderServer(t, reqAssert)
	defer srv.Close()
	cfg := auth.ProviderCfg{
		TokenURL:    srv.URL + "/oauth2/auth",
		ClientID:    "client id 0",
		Secret:      "secure",
		RedirectURI: "http://localhost",
	}
	providers := auth.NewProviders(NewTestProvider(cfg, client))
	svc := auth.New(staticGenerator, providers)
	expected := auth.User{
		ID:    "1",
		Email: "test@localhost",
	}

	token, user, err := svc.Authenticate(context.Background(), "test", "code-0", "someVerifier")

	require.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, expected, user)
}

func TestAuthenticate_ExchangeError(t *testing.T) {
	srv := startProviderServer(t, nil)
	defer srv.Close()
	cfg := auth.ProviderCfg{
		TokenURL: srv.URL + "/oauth2/autherror",
	}
	providers := auth.NewProviders(NewTestProvider(cfg, client))
	svc := auth.New(staticGenerator, providers)

	_, _, err := svc.Authenticate(context.Background(), "test", "", "")

	require.Error(t, err)
	var appErr aerrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, aerrors.ErrUnknown, appErr.ErrorType)
	assert.Equal(t, "exchange-code-failed", appErr.Key)
}

func TesRevoke(t *testing.T) {
	ctx := context.Background()
	srv := startProviderServer(t, func(uri, body string) {
		expectedBody := url.Values{
			"token": {"token-0"},
		}
		if strings.HasSuffix(uri, "/revoke") {
			assert.Equal(t, expectedBody.Encode(), body)
		}
	})
	defer srv.Close()
	cfg := auth.ProviderCfg{
		RevokeURL: srv.URL + "/oauth2/revoke",
	}
	providers := auth.NewProviders(NewTestProvider(cfg, client))
	svc := auth.New(nil, providers)

	err := svc.Revoke(ctx, &auth.JWT{AccessToken: "token-0", Provider: "test"})

	require.NoError(t, err)
}

func TestRevoke_Error(t *testing.T) {
	ctx := context.Background()
	srv := startProviderServer(t, nil)
	defer srv.Close()
	cfg := auth.ProviderCfg{
		RevokeURL: srv.URL + "/oauth2/revokeerr",
	}
	providers := auth.NewProviders(NewTestProvider(cfg, client))
	svc := auth.New(nil, providers)

	err := svc.Revoke(ctx, &auth.JWT{AccessToken: "token-0", Provider: "test"})

	require.Error(t, err)
	var appErr aerrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, aerrors.ErrUnknown, appErr.ErrorType)
	assert.Equal(t, "revoke-token-failed", appErr.Key)
}

func startProviderServer(t *testing.T, bodyAssert func(url, body string)) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if bodyAssert != nil {
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			bodyAssert(r.RequestURI, string(body))
		}

		if r.Method == http.MethodPost && strings.HasSuffix(r.RequestURI, "/auth") {
			_, err := w.Write(test.ToJSON(t, auth.JWT{}))
			assert.NoError(t, err)

			return
		}

		if r.Method == http.MethodPost && strings.HasSuffix(r.RequestURI, "/revoke") {
			w.WriteHeader(http.StatusOK)

			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}))
}

func NewTestProvider(cfg auth.ProviderCfg, client *http.Client) auth.OIDCProvider {
	return auth.OIDCProvider{
		Name:        "test",
		AuthURL:     cfg.AuthURL,
		TokenURL:    cfg.TokenURL,
		RevokeURL:   cfg.RevokeURL,
		RedirectURI: cfg.RedirectURI,
		Client:      client,
		ClientID:    cfg.ClientID,
		Secret:      cfg.Secret,
		Scopes:      cfg.Scopes,
		Validate: func(ctx context.Context, token *auth.JWT, clientID string) (auth.Claim, error) {
			return auth.NewClaim("1", "test@localhost"), nil
		},
	}
}
