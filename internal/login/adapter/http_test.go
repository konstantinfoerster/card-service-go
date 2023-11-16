package adapter_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	"github.com/konstantinfoerster/card-service-go/internal/common/problemjson"
	"github.com/konstantinfoerster/card-service-go/internal/common/server"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	loginadapter "github.com/konstantinfoerster/card-service-go/internal/login/adapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var staticTimeSvc = common.NewFakeTimeService(time.Now())

const cookieEncryptionKey = "01234567890123456789012345678901"

type mockOIDCService struct {
	FakeGetAuthURL           func(provider string) (*oidc.RedirectURL, error)
	FakeAuthenticate         func(provider string, authCode string) (*auth.User, *oidc.JSONWebToken, error)
	FakeGetAuthenticatedUser func(provider string, token *oidc.JSONWebToken) (*auth.User, error)
	FakeLogout               func(provider string, token *oidc.JSONWebToken) error
}

var _ oidc.Service = (*mockOIDCService)(nil)

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
				URL:   "http://authserver.local",
				State: "state-0",
			}, nil
		},
	}
	srv := defaultServer(svc, staticTimeSvc)
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/login/myProvider"),
	)
	expectedCookie := &http.Cookie{
		Name:    "TOKEN_STATE",
		Value:   "state-0",
		Expires: expiresIn(5 * time.Second),
	}

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, "http://authserver.local", resp.Header.Get("Location"))
	assertCookie(t, expectedCookie, resp.Cookies()[0])
}

func TestGetLoginURLError(t *testing.T) {
	svc := &mockOIDCService{
		FakeGetAuthURL: func(provider string) (*oidc.RedirectURL, error) {
			return nil, fmt.Errorf("some error")
		},
	}
	srv := defaultServer(svc, staticTimeSvc)
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/login/myProvider"),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assertErrorResponse(t, resp, http.StatusInternalServerError)
}

func TestPostExchangeCode(t *testing.T) {
	sessionExpiresIn := 100
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
				ExpiresIn: sessionExpiresIn,
			}

			return u, token, nil
		},
	}
	srv := defaultServer(svc, staticTimeSvc)
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodPost),
		commontest.WithURL("http://localhost/login/myProvider/token"),
		commontest.WithJSONBody(t, loginadapter.AuthCode{Code: "myAuthCode", State: "state-0"}),
		commontest.WithCookie(&http.Cookie{
			Name:  "TOKEN_STATE",
			Value: encryptCookieValue(t, "state-0"),
		}),
	)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{ExpiresIn: sessionExpiresIn})
	expectedSessionCookie := &http.Cookie{
		Name:    "SESSION",
		Value:   token,
		Expires: expiresIn(time.Duration(sessionExpiresIn) * time.Second),
	}
	expectedTokenCookie := &http.Cookie{
		Name:    "TOKEN_STATE",
		Value:   "invalid",
		Expires: expiresIn(-7 * 24 * time.Hour),
	}

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, fiber.MIMEApplicationJSONCharsetUTF8, resp.Header.Get(fiber.HeaderContentType))
	assert.Equal(t, &loginadapter.User{Username: "myUser"}, commontest.FromJSON[loginadapter.User](t, resp))
	assert.Len(t, resp.Cookies(), 2)
	assertCookie(t, expectedTokenCookie, resp.Cookies()[0])
	assertCookie(t, expectedSessionCookie, resp.Cookies()[1])
}

func TestPostExchangeInvalidInput(t *testing.T) {
	cases := []struct {
		name       string
		body       *loginadapter.AuthCode
		stateValue string
		svc        oidc.Service
		statusCode int
	}{
		{
			name:       "No state cookie",
			body:       &loginadapter.AuthCode{Code: "myAuthCode", State: "state-0"},
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
			body:       &loginadapter.AuthCode{Code: "", State: ""},
			stateValue: "state-0",
			svc:        &mockOIDCService{},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "state mismatch",
			body:       &loginadapter.AuthCode{Code: "myAuthCode", State: "state-1"},
			stateValue: "state-0",
			svc:        &mockOIDCService{},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Failed authentication",
			body:       &loginadapter.AuthCode{Code: "myAuthCode", State: "state-0"},
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
			srv := defaultServer(tc.svc, staticTimeSvc)
			req := commontest.NewRequest(
				commontest.WithMethod(http.MethodPost),
				commontest.WithURL("http://localhost/login/myProvider/token"),
				commontest.WithJSONBody(t, tc.body),
			)
			if tc.stateValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  "TOKEN_STATE",
					Value: encryptCookieValue(t, tc.stateValue),
				})
			}

			resp, err := srv.Test(req)
			defer commontest.Close(t, resp)

			require.NoError(t, err)
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
	srv := defaultServer(svc, staticTimeSvc)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/user"),
		commontest.WithCookie(&http.Cookie{
			Name:  "SESSION",
			Value: encryptCookieValue(t, token),
		}),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, fiber.MIMEApplicationJSONCharsetUTF8, resp.Header.Get(fiber.HeaderContentType))
	assert.Equal(t, &loginadapter.User{Username: "myUser"}, commontest.FromJSON[loginadapter.User](t, resp))
}

func TestGetCurrentUserNotLoggedIn(t *testing.T) {
	srv := defaultServer(&mockOIDCService{}, staticTimeSvc)
	req := httptest.NewRequest(http.MethodGet, "http://localhost/user", nil)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assertErrorResponse(t, resp, http.StatusUnauthorized)
}

func TestLogout(t *testing.T) {
	svc := &mockOIDCService{
		FakeLogout: func(provider string, token *oidc.JSONWebToken) error {
			return nil
		},
	}
	srv := defaultServer(svc, staticTimeSvc)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{Provider: "myProvider"})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodPost),
		commontest.WithURL("http://localhost/logout"),
		commontest.WithCookie(&http.Cookie{
			Name:  "SESSION",
			Value: encryptCookieValue(t, token),
		}),
	)
	expectedSessionCookie := &http.Cookie{
		Name:    "SESSION",
		Value:   "invalid",
		Expires: expiresIn(-7 * 24 * time.Hour),
	}

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, resp.Cookies(), 1)
	assertCookie(t, expectedSessionCookie, resp.Cookies()[0])
}

func TestLogoutNoSession(t *testing.T) {
	srv := defaultServer(&mockOIDCService{}, staticTimeSvc)
	req := httptest.NewRequest(http.MethodPost, "http://localhost/logout", nil)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
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
			srv := defaultServer(tc.src, staticTimeSvc)
			req := httptest.NewRequest(http.MethodPost, "http://localhost/logout", nil)
			value := tc.sessionValueFn(t)
			req.AddCookie(&http.Cookie{
				Name:  "SESSION",
				Value: encryptCookieValue(t, value),
			})

			resp, err := srv.Test(req)
			defer commontest.Close(t, resp)

			require.NoError(t, err)
			assertErrorResponse(t, resp, tc.statusCode)
		})
	}
}

func assertErrorResponse(t *testing.T, resp *http.Response, expectedStatus int) {
	t.Helper()

	assert.Equal(t, expectedStatus, resp.StatusCode)
	assert.Equal(t, problemjson.ContentType, resp.Header.Get(fiber.HeaderContentType))

	result := commontest.FromJSON[problemjson.ProblemJSON](t, resp)
	assert.Equal(t, expectedStatus, result.Status)
}

func assertCookie(t *testing.T, expected *http.Cookie, actual *http.Cookie) {
	t.Helper()

	// these properties should always be set like this
	expected.HttpOnly = true
	expected.Path = "/api"
	expected.Secure = true
	expected.SameSite = http.SameSiteStrictMode

	actual.Value = decryptCookieValue(t, actual.Value)
	actual.Raw = ""
	actual.RawExpires = ""
	assert.Equal(t, expected, actual)
}

func decryptCookieValue(t *testing.T, value string) string {
	t.Helper()

	v, err := encryptcookie.DecryptCookie(value, cookieEncryptionKey)
	require.NoError(t, err)

	return v
}

func encryptCookieValue(t *testing.T, value string) string {
	t.Helper()

	v, err := encryptcookie.EncryptCookie(value, cookieEncryptionKey)
	require.NoError(t, err)

	return v
}

func expiresIn(d time.Duration) time.Time {
	return staticTimeSvc.Now().Add(d).Truncate(time.Second).UTC()
}

func defaultServer(svc oidc.Service, timeSvc common.TimeService) *server.Server {
	cfg := &config.Server{
		Cookie: config.Cookie{
			EncryptionKey: cookieEncryptionKey,
		},
	}
	srv := server.NewHTTPServer(cfg)

	cfgOidc := config.Oidc{
		StateCookieAge:    5 * time.Second,
		SessionCookieName: "SESSION",
		RedirectURI:       "http://localhost/home",
	}
	srv.RegisterAPIRoutes(func(r fiber.Router) {
		loginadapter.Routes(r.Group("/"), cfgOidc, svc, timeSvc)
	})

	return srv
}
