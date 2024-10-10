package auth_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var client = &http.Client{}

func TestUnsupportedProvider(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		errType  aerrors.ErrorType
	}{
		{
			name:     "Unknown provider",
			provider: "unknown",
			errType:  aerrors.ErrTypeInvalidInput,
		},
		{
			name:     "empty provider",
			provider: "",
			errType:  aerrors.ErrTypeInvalidInput,
		},
		{
			name:     "space only provider",
			provider: "  ",
			errType:  aerrors.ErrTypeInvalidInput,
		},
	}

	for _, tc := range cases {
		t.Run("Authenticate - "+tc.name, func(t *testing.T) {
			ctx := context.Background()
			svc := auth.New(config.Oidc{}, auth.Providers{})

			_, _, err := svc.Authenticate(ctx, tc.provider, "")

			var appErr aerrors.AppError
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, tc.errType, appErr.ErrorType)
		})

		t.Run("AuthURL - "+tc.name, func(t *testing.T) {
			svc := auth.New(config.Oidc{}, auth.Providers{})

			_, err := svc.AuthURL(tc.provider)

			var appErr aerrors.AppError
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, tc.errType, appErr.ErrorType)
		})

		t.Run("AuthInfo - "+tc.name, func(t *testing.T) {
			ctx := context.Background()
			svc := auth.New(config.Oidc{}, auth.Providers{})

			_, err := svc.AuthInfo(ctx, tc.provider, nil)

			var appErr aerrors.AppError
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, tc.errType, appErr.ErrorType)
		})

		t.Run("Logout - "+tc.name, func(t *testing.T) {
			ctx := context.Background()
			svc := auth.New(config.Oidc{}, auth.Providers{})

			err := svc.Logout(ctx, &auth.JWT{Provider: tc.provider})

			var appErr aerrors.AppError
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, tc.errType, appErr.ErrorType)
		})
	}
}

func TestAuthURL(t *testing.T) {
	pCfg := config.Provider{
		AuthURL:     "http://localhost/oauth2/auth",
		RedirectURI: "http://localhost",
		ClientID:    "client id 0",
		Scope:       "openid email",
	}
	svc := auth.New(config.Oidc{}, auth.NewProviders(auth.TestProvider(pCfg, client)))

	actualURL, err := svc.AuthURL("test")
	expectedURL := "http://localhost/oauth2/auth?state=" + actualURL.State + "&client_id=client+id+0&redirect_uri=http%3A%2F%2Flocalhost&scope=openid+email&response_type=code&access_type=offline"

	require.NoError(t, err)
	assert.NotEmpty(t, strings.TrimSpace(actualURL.State))
	assert.Equal(t, expectedURL, actualURL.URL)
}

func TestAuthenticate(t *testing.T) {
	ctx := context.Background()
	expectedBody := url.Values{
		"code":          {"code-0"},
		"client_id":     {"client id 0"},
		"client_secret": {"secure"},
		"redirect_uri":  {"http://localhost"},
		"grant_type":    {"authorization_code"},
	}
	srv := startProviderServer(t, expectedBody.Encode())
	defer srv.Close()
	pCfg := config.Provider{
		TokenURL:    srv.URL + "/oauth2/auth",
		ClientID:    "client id 0",
		Secret:      "secure",
		RedirectURI: "http://localhost",
	}
	svc := auth.New(config.Oidc{}, auth.NewProviders(auth.TestProvider(pCfg, client)))

	user, token, err := svc.Authenticate(ctx, "test", "code-0")

	require.NoError(t, err)
	assert.Equal(t, "test", token.Provider)
	assert.Equal(t, auth.NewClaims("1", "test@localhost"), user)
}

func TestAuthenticateOidcServerError(t *testing.T) {
	ctx := context.Background()
	srv := startProviderServer(t, "")
	defer srv.Close()
	pCfg := config.Provider{
		TokenURL: srv.URL + "/oauth2/autherror",
	}
	svc := auth.New(config.Oidc{}, auth.NewProviders(auth.TestProvider(pCfg, client)))

	_, _, err := svc.Authenticate(ctx, "test", "code-0")
	require.Error(t, err)

	var appErr aerrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, aerrors.ErrTypeUnknown, appErr.ErrorType)
}

func TestAuthInfo(t *testing.T) {
	ctx := context.Background()
	svc := auth.New(config.Oidc{}, auth.NewProviders(auth.TestProvider(config.Provider{}, client)))

	user, err := svc.AuthInfo(ctx, "test", &auth.JWT{})

	require.NoError(t, err)
	assert.Equal(t, auth.NewClaims("1", "test@localhost"), user)
}

func TestLogout(t *testing.T) {
	ctx := context.Background()
	expectedBody := url.Values{
		"token": {"token-0"},
	}
	srv := startProviderServer(t, expectedBody.Encode())
	defer srv.Close()
	pCfg := config.Provider{
		RevokeURL: srv.URL + "/oauth2/revoke",
	}
	svc := auth.New(config.Oidc{}, auth.NewProviders(auth.TestProvider(pCfg, client)))

	err := svc.Logout(ctx, &auth.JWT{AccessToken: "token-0", Provider: "test"})

	require.NoError(t, err)
}

func startProviderServer(t *testing.T, expectedBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := 500

		if r.Method == http.MethodPost && strings.HasSuffix(r.RequestURI, "/auth") {
			_, err := w.Write(test.ToJSON(t, auth.JWT{}))
			assert.NoError(t, err)

			status = 200
		}

		if r.Method == http.MethodPost && strings.HasSuffix(r.RequestURI, "/revoke") {
			status = 200
		}

		if expectedBody != "" {
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, expectedBody, string(body))
		}

		w.WriteHeader(status)
	}))
}
