package oidc_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var client = &http.Client{
	Timeout: time.Second * time.Duration(5),
}

func TestUnsupportedProvider(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		errType  common.ErrorType
	}{
		{
			name:     "Unknown provider",
			provider: "unknown",
			errType:  common.ErrTypeInvalidInput,
		},
		{
			name:     "empty provider",
			provider: "",
			errType:  common.ErrTypeInvalidInput,
		},
		{
			name:     "empty provider",
			provider: "  ",
			errType:  common.ErrTypeInvalidInput,
		},
	}

	for _, tc := range cases {
		t.Run("Authenticate - "+tc.name, func(t *testing.T) {
			svc := oidc.New(config.Oidc{}, []oidc.Provider{oidc.TestProvider(config.Provider{}, nil)})

			_, _, err := svc.Authenticate(tc.provider, "")

			var appErr common.AppError
			assert.ErrorAs(t, err, &appErr)
			assert.Equal(t, tc.errType, appErr.ErrorType)
		})

		t.Run("GetAuthURL - "+tc.name, func(t *testing.T) {
			svc := oidc.New(config.Oidc{}, []oidc.Provider{oidc.TestProvider(config.Provider{}, nil)})

			_, err := svc.GetAuthURL(tc.provider)

			var appErr common.AppError
			assert.ErrorAs(t, err, &appErr)
			assert.Equal(t, tc.errType, appErr.ErrorType)
		})

		t.Run("GetAuthenticatedUser - "+tc.name, func(t *testing.T) {
			svc := oidc.New(config.Oidc{}, []oidc.Provider{oidc.TestProvider(config.Provider{}, nil)})

			_, err := svc.GetAuthenticatedUser(tc.provider, nil)

			var appErr common.AppError
			assert.ErrorAs(t, err, &appErr)
			assert.Equal(t, tc.errType, appErr.ErrorType)
		})

		t.Run("Logout - "+tc.name, func(t *testing.T) {
			svc := oidc.New(config.Oidc{}, []oidc.Provider{oidc.TestProvider(config.Provider{}, nil)})

			err := svc.Logout(tc.provider, nil)

			var appErr common.AppError
			assert.ErrorAs(t, err, &appErr)
			assert.Equal(t, tc.errType, appErr.ErrorType)
		})
	}
}

func TestGetAuthURL(t *testing.T) {
	cfg := config.Oidc{
		RedirectURI: "http://localhost",
	}
	pCfg := config.Provider{
		AuthURL:  "http://localhost/oauth2/auth",
		ClientID: "client id 0",
		Scope:    "openid email",
	}
	svc := oidc.New(cfg, []oidc.Provider{oidc.TestProvider(pCfg, client)})

	actualURL, err := svc.GetAuthURL("test")
	expectedURL := "http://localhost/oauth2/auth?state=" + actualURL.State + "&client_id=client+id+0&redirect_uri=http%3A%2F%2Flocalhost&scope=openid+email&response_type=code&access_type=offline"

	assert.NoError(t, err)
	assert.NotEmpty(t, strings.TrimSpace(actualURL.State))
	assert.Equal(t, expectedURL, actualURL.URL)
}

func TestAuthenticate(t *testing.T) {
	expectedBody := url.Values{
		"code":          {"code-0"},
		"client_id":     {"client id 0"},
		"client_secret": {"secure"},
		"redirect_uri":  {"http://localhost"},
		"grant_type":    {"authorization_code"},
	}
	srv := startProviderServer(t, expectedBody.Encode())
	defer srv.Close()
	cfg := config.Oidc{
		RedirectURI: "http://localhost",
	}
	pCfg := config.Provider{
		TokenURL: fmt.Sprintf("%s/oauth2/auth", srv.URL),
		ClientID: "client id 0",
		Secret:   "secure",
	}
	svc := oidc.New(cfg, []oidc.Provider{oidc.TestProvider(pCfg, client)})

	user, token, err := svc.Authenticate("test", "code-0")

	assert.NoError(t, err)
	assert.Equal(t, "test", token.Provider)
	assert.Equal(t, &auth.User{ID: "1", Username: "test@localhost"}, user)
}

func TestAuthenticateOidcServerError(t *testing.T) {
	srv := startProviderServer(t, "")
	defer srv.Close()
	pCfg := config.Provider{
		TokenURL: fmt.Sprintf("%s/oauth2/autherror", srv.URL),
	}
	svc := oidc.New(config.Oidc{}, []oidc.Provider{oidc.TestProvider(pCfg, client)})

	_, _, err := svc.Authenticate("test", "code-0")
	assert.Error(t, err)
	var appErr common.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, common.ErrTypeUnknown, appErr.ErrorType)
}

func TestGetAuthenticatedUser(t *testing.T) {
	svc := oidc.New(config.Oidc{}, []oidc.Provider{oidc.TestProvider(config.Provider{}, client)})

	user, err := svc.GetAuthenticatedUser("test", &oidc.JSONWebToken{})

	assert.NoError(t, err)
	assert.Equal(t, &auth.User{ID: "1", Username: "test@localhost"}, user)
}

func TestLogout(t *testing.T) {
	expectedBody := url.Values{
		"token": {"token-0"},
	}
	srv := startProviderServer(t, expectedBody.Encode())
	defer srv.Close()
	pCfg := config.Provider{
		RevokeURL: fmt.Sprintf("%s/oauth2/revoke", srv.URL),
	}
	svc := oidc.New(config.Oidc{}, []oidc.Provider{oidc.TestProvider(pCfg, client)})

	err := svc.Logout("test", &oidc.JSONWebToken{AccessToken: "token-0"})

	assert.NoError(t, err)
}

func startProviderServer(t *testing.T, expectedBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := 500

		if r.Method == http.MethodPost && strings.HasSuffix(r.RequestURI, "/auth") {
			_, err := w.Write(commontest.ToJSON(t, oidc.JSONWebToken{}))
			require.NoError(t, err)

			status = 200
		}

		if r.Method == http.MethodPost && strings.HasSuffix(r.RequestURI, "/revoke") {
			status = 200
		}

		if expectedBody != "" {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Equal(t, expectedBody, string(body))
		}

		w.WriteHeader(status)
	}))
}
