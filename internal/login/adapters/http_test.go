package adapters_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/problemjson"
	"github.com/konstantinfoerster/card-service-go/internal/common/server"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	loginadapters "github.com/konstantinfoerster/card-service-go/internal/login/adapters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOIDCService struct {
	FakeGetAuthURL           func(provider string) (*oidc.RedirectURL, error)
	FakeAuthenticate         func(provider string, authCode string) (*auth.User, *oidc.JSONWebToken, error)
	FakeGetAuthenticatedUser func(provider string, token *oidc.JSONWebToken) (*auth.User, error)
	FakeLogout               func(provider string, token *oidc.JSONWebToken) error
}

func (s *mockOIDCService) GetAuthURL(provider string) (*oidc.RedirectURL, error) {
	if s.FakeGetAuthURL != nil {
		return s.FakeGetAuthURL(provider)
	}

	return nil, fmt.Errorf("unexpected function call, no fake implementation provided")
}

func (s *mockOIDCService) Authenticate(provider string, authCode string) (*auth.User, *oidc.JSONWebToken, error) {
	if s.FakeAuthenticate != nil {
		return s.FakeAuthenticate(provider, authCode)
	}

	return nil, nil, fmt.Errorf("unexpected function call, no fake implementation provided")
}

func (s *mockOIDCService) GetAuthenticatedUser(provider string, token *oidc.JSONWebToken) (*auth.User, error) {
	if s.FakeGetAuthenticatedUser != nil {
		return s.FakeGetAuthenticatedUser(provider, token)
	}

	return nil, fmt.Errorf("unexpected function call, no fake implementation provided")
}

func (s *mockOIDCService) Logout(provider string, token *oidc.JSONWebToken) error {
	if s.FakeLogout != nil {
		return s.FakeLogout(provider, token)
	}

	return fmt.Errorf("unexpected function call, no fake implementation provided")
}

func TestGetLoginURL(t *testing.T) {
	svc := &mockOIDCService{
		FakeGetAuthURL: func(provider string) (*oidc.RedirectURL, error) {
			if provider != "myProvider" {
				return nil, fmt.Errorf("invalid provider %s", provider)
			}

			return &oidc.RedirectURL{
				URL:   "https://authserver.local",
				State: "state-0",
			}, nil
		},
	}
	srv := defaultServer(svc)
	expectedCookie := &http.Cookie{
		Name:     "TOKEN_STATE",
		Value:    "state-0",
		HttpOnly: true,
		Path:     "/",
		MaxAge:   5,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	req := httptest.NewRequest(http.MethodGet, "https://localhost/login/myProvider", nil)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, "https://authserver.local", resp.Header.Get("Location"))
	assertCookie(t, srv.Cfg.Cookie.EncryptionKey, expectedCookie, resp.Cookies()[0])
}

func TestGetLoginURLError(t *testing.T) {
	svc := &mockOIDCService{
		FakeGetAuthURL: func(provider string) (*oidc.RedirectURL, error) {
			return nil, fmt.Errorf("some error")
		},
	}
	srv := defaultServer(svc)
	req := httptest.NewRequest(http.MethodGet, "https://localhost/login/myProvider", nil)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	assert.NoError(t, err)
	assertErrorResponse(t, resp, http.StatusInternalServerError)
}

func TestPostExchangeCode(t *testing.T) {
	svc := &mockOIDCService{
		FakeAuthenticate: func(provider string, authCode string) (*auth.User, *oidc.JSONWebToken, error) {
			if provider != "myProvider" {
				return nil, nil, fmt.Errorf("invalid provider %s", provider)
			}
			if authCode != "myAuthCode" {
				return nil, nil, fmt.Errorf("invalid auth code %s", authCode)
			}

			u := &auth.User{
				Username: "myUser",
			}
			token := &oidc.JSONWebToken{
				ExpiresIn: 5,
			}

			return u, token, nil
		},
	}
	srv := defaultServer(svc)
	body := bytes.NewReader(commontest.ToJSON(t, loginadapters.AuthCode{Code: "myAuthCode", State: "state-0"}))
	req := httptest.NewRequest(http.MethodPost, "https://localhost/login/myProvider/token", body)
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  "TOKEN_STATE",
		Value: encryptCookieValue(t, srv.Cfg.Cookie.EncryptionKey, "state-0"),
	})
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{ExpiresIn: 5})
	expectedSessionCookie := &http.Cookie{
		Name:     "SESSION",
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   10,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	expectedTokenCookie := &http.Cookie{
		Name:     "TOKEN_STATE",
		Value:    "invalid",
		HttpOnly: true,
		Path:     "/",
		MaxAge:   0,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, fiber.MIMEApplicationJSONCharsetUTF8, resp.Header.Get(fiber.HeaderContentType))
	assert.Equal(t, &loginadapters.User{Username: "myUser"}, commontest.FromJSON[loginadapters.User](t, resp))
	assert.Len(t, resp.Cookies(), 2)
	assertCookie(t, srv.Cfg.Cookie.EncryptionKey, expectedTokenCookie, resp.Cookies()[0])
	assertCookie(t, srv.Cfg.Cookie.EncryptionKey, expectedSessionCookie, resp.Cookies()[1])
}

func TestPostExchangeInvalidInput(t *testing.T) {
	cases := []struct {
		name       string
		body       io.Reader
		stateValue string
		svc        oidc.Service
		statusCode int
	}{
		{
			name:       "No state cookie",
			body:       bytes.NewReader(commontest.ToJSON(t, loginadapters.AuthCode{Code: "myAuthCode", State: "state-0"})),
			stateValue: "",
			svc:        &mockOIDCService{},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "No body",
			body:       nil,
			stateValue: "state-0",
			svc:        &mockOIDCService{},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Invalid body",
			body:       bytes.NewReader(commontest.ToJSON(t, loginadapters.AuthCode{Code: "", State: ""})),
			stateValue: "state-0",
			svc:        &mockOIDCService{},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "state mismatch",
			body:       bytes.NewReader(commontest.ToJSON(t, loginadapters.AuthCode{Code: "myAuthCode", State: "state-1"})),
			stateValue: "state-0",
			svc:        &mockOIDCService{},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Failed authentication",
			body:       bytes.NewReader(commontest.ToJSON(t, loginadapters.AuthCode{Code: "myAuthCode", State: "state-0"})),
			stateValue: "state-0",
			svc: &mockOIDCService{
				FakeAuthenticate: func(provider string, authCode string) (*auth.User, *oidc.JSONWebToken, error) {
					return nil, nil, fmt.Errorf("some error")
				},
			},
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := defaultServer(tc.svc)
			req := httptest.NewRequest(http.MethodPost, "https://localhost/login/myProvider/token", tc.body)
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			if tc.stateValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  "TOKEN_STATE",
					Value: encryptCookieValue(t, srv.Cfg.Cookie.EncryptionKey, tc.stateValue),
				})
			}

			resp, err := srv.Test(req)
			defer commontest.Close(t, resp)

			assert.NoError(t, err)
			assertErrorResponse(t, resp, tc.statusCode)
		})
	}
}

func TestGetCurrentUser(t *testing.T) {
	svc := &mockOIDCService{
		FakeGetAuthenticatedUser: func(provider string, token *oidc.JSONWebToken) (*auth.User, error) {
			return &auth.User{
				Username: "myUser",
			}, nil
		},
	}
	srv := defaultServer(svc)
	req := httptest.NewRequest(http.MethodGet, "https://localhost/user", nil)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{})
	req.AddCookie(&http.Cookie{
		Name:     "SESSION",
		Value:    encryptCookieValue(t, srv.Cfg.Cookie.EncryptionKey, token),
		HttpOnly: true,
		Path:     "/",
		MaxAge:   0,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, fiber.MIMEApplicationJSONCharsetUTF8, resp.Header.Get(fiber.HeaderContentType))
	assert.Equal(t, &loginadapters.User{Username: "myUser"}, commontest.FromJSON[loginadapters.User](t, resp))
}

func TestGetCurrentUserNotLoggedIn(t *testing.T) {
	srv := defaultServer(&mockOIDCService{})
	req := httptest.NewRequest(http.MethodGet, "https://localhost/user", nil)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	assert.NoError(t, err)
	assertErrorResponse(t, resp, http.StatusUnauthorized)
}

func TestLogout(t *testing.T) {
	svc := &mockOIDCService{
		FakeLogout: func(provider string, token *oidc.JSONWebToken) error {
			return nil
		},
	}
	srv := defaultServer(svc)
	req := httptest.NewRequest(http.MethodPost, "https://localhost/logout", nil)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{Provider: "myProvider"})
	req.AddCookie(&http.Cookie{
		Name:     "SESSION",
		Value:    encryptCookieValue(t, srv.Cfg.Cookie.EncryptionKey, token),
		HttpOnly: true,
		Path:     "/",
		MaxAge:   0,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	expectedSessionCookie := &http.Cookie{
		Name:     "SESSION",
		Value:    "invalid",
		HttpOnly: true,
		Path:     "/",
		MaxAge:   0,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, resp.Cookies(), 1)
	assertCookie(t, srv.Cfg.Cookie.EncryptionKey, expectedSessionCookie, resp.Cookies()[0])
}

func TestLogoutNoSession(t *testing.T) {
	srv := defaultServer(&mockOIDCService{})
	req := httptest.NewRequest(http.MethodPost, "https://localhost/logout", nil)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLogoutError(t *testing.T) {
	cases := []struct {
		name           string
		sessionValueFn func(t *testing.T) string
		statusCode     int
		src            oidc.Service
	}{
		{
			name: "Invalid session value",
			sessionValueFn: func(t *testing.T) string {
				return "123"
			},
			statusCode: http.StatusBadRequest,
			src:        &mockOIDCService{},
		},
		{
			name: "Session without provider",
			sessionValueFn: func(t *testing.T) string {
				return commontest.Base64Encoded(t, &oidc.JSONWebToken{})
			},
			statusCode: http.StatusBadRequest,
			src:        &mockOIDCService{},
		},
		{
			name: "Failed logout",
			sessionValueFn: func(t *testing.T) string {
				return commontest.Base64Encoded(t, &oidc.JSONWebToken{Provider: "myProvider"})
			},
			statusCode: http.StatusInternalServerError,
			src: &mockOIDCService{
				FakeLogout: func(provider string, token *oidc.JSONWebToken) error {
					return fmt.Errorf("some error")
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := defaultServer(tc.src)
			req := httptest.NewRequest(http.MethodPost, "https://localhost/logout", nil)
			value := tc.sessionValueFn(t)
			req.AddCookie(&http.Cookie{
				Name:  "SESSION",
				Value: encryptCookieValue(t, srv.Cfg.Cookie.EncryptionKey, value),
			})

			resp, err := srv.Test(req)
			defer commontest.Close(t, resp)

			assert.NoError(t, err)
			assertErrorResponse(t, resp, tc.statusCode)
		})
	}
}

func assertErrorResponse(t *testing.T, resp *http.Response, expectedStatus int) {
	assert.Equal(t, expectedStatus, resp.StatusCode)
	assert.Contains(t, resp.Header.Get(fiber.HeaderContentType), "application/json")

	result := commontest.FromJSON[problemjson.ProblemJSON](t, resp)
	assert.Equal(t, expectedStatus, result.Status)
}

func assertCookie(t *testing.T, encKey string, expected *http.Cookie, actual *http.Cookie) {
	actual.Value = decryptCookieValue(t, encKey, actual.Value)
	actual.Raw = ""
	actual.RawExpires = ""
	assert.Equal(t, expected, actual)
}

func decryptCookieValue(t *testing.T, encKey string, value string) string {
	t.Helper()

	v, err := encryptcookie.DecryptCookie(value, encKey)
	require.NoError(t, err)

	return v
}

func encryptCookieValue(t *testing.T, encKey string, value string) string {
	t.Helper()

	v, err := encryptcookie.EncryptCookie(value, encKey)
	require.NoError(t, err)

	return v
}

func defaultServer(service oidc.Service) *server.Server {
	cfg := &config.Server{
		Cookie: config.Cookie{
			EncryptionKey: "01234567890123456789012345678901",
		},
	}
	srv := server.NewHTTPServer(cfg)

	cfgOidc := config.Oidc{
		StateCookieAge:    time.Second * 5,
		SessionCookieName: "SESSION",
		RedirectURI:       "https://localhost/home",
	}
	srv.RegisterAPIRoutes(func(r fiber.Router) {
		loginadapters.Routes(r.Group("/"), cfgOidc, service)
	})

	return srv
}
