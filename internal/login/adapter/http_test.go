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
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
	"github.com/konstantinfoerster/card-service-go/internal/common/problemjson"
	"github.com/konstantinfoerster/card-service-go/internal/common/server"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	loginadapter "github.com/konstantinfoerster/card-service-go/internal/login/adapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var staticTimeSvc = common.NewFakeTimeService(time.Now())

var cookieEncryptionKey = ""

type mockOIDCService struct {
	FakeGetAuthURL           func(provider string) (*oidc.RedirectURL, error)
	FakeAuthenticate         func(provider string, authCode string) (*auth.User, *oidc.JSONWebToken, error)
	FakeGetAuthenticatedUser func(provider string, token *oidc.JSONWebToken) (*auth.User, error)
	FakeLogout               func(token *oidc.JSONWebToken) error
	EncryptionKey            string
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

func (s *mockOIDCService) Logout(token *oidc.JSONWebToken) error {
	if s.FakeLogout != nil {
		return s.FakeLogout(token)
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
	srv := loginServer(svc, staticTimeSvc)
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/login/myProvider"),
	)
	expectedCookie := &http.Cookie{
		Name:     "TOKEN_STATE",
		Value:    "state-0",
		Expires:  expiresIn(5 * time.Second),
		SameSite: http.SameSiteLaxMode,
	}

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	require.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, "http://authserver.local", resp.Header.Get("Location"))
	assertCookie(t, expectedCookie, resp.Cookies()[0])
}

func TestGetLoginURLError(t *testing.T) {
	svc := &mockOIDCService{
		FakeGetAuthURL: func(provider string) (*oidc.RedirectURL, error) {
			return nil, fmt.Errorf("some error")
		},
	}
	srv := loginServer(svc, staticTimeSvc)
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/login/myProvider"),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assertErrorResponse(t, resp, http.StatusInternalServerError)
}

func TestExchangeCode(t *testing.T) {
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
				Provider:  provider,
			}

			return u, token, nil
		},
	}
	srv := loginServer(svc, staticTimeSvc)
	rawState, err := oidc.NewState("myProvider").Encode()
	require.NoError(t, err)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{ExpiresIn: sessionExpiresIn, Provider: "myProvider"})
	expectedSessionCookie := &http.Cookie{
		Name:     "SESSION",
		Value:    token,
		Expires:  expiresIn(time.Duration(sessionExpiresIn) * time.Second),
		SameSite: http.SameSiteStrictMode,
	}
	expectedTokenCookie := &http.Cookie{
		Name:     "TOKEN_STATE",
		Value:    "invalid",
		Expires:  expiresIn(-7 * 24 * time.Hour),
		SameSite: http.SameSiteStrictMode,
	}

	cases := []struct {
		name                string
		acceptHeader        string
		expectedContentType string
		expectedStatus      int
		expectedBodyPart    []byte
	}{
		{
			name:                "html response",
			acceptHeader:        fiber.MIMETextHTMLCharsetUTF8,
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			expectedStatus:      http.StatusOK,
			expectedBodyPart:    []byte("<meta http-equiv=\"Refresh\" content=\"0; url='/'\"/>"),
		},
		{
			name:                "json response",
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			expectedStatus:      http.StatusOK,
			expectedBodyPart:    commontest.ToJSON(t, &commonhttp.ClientUser{Username: "myUser", Initials: "my"}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := commontest.NewRequest(
				commontest.WithMethod(http.MethodGet),
				commontest.WithURL(fmt.Sprintf("http://localhost/login/callback?code=myAuthCode&state=%s", rawState)),
				commontest.WithCookie(&http.Cookie{
					Name:  "TOKEN_STATE",
					Value: encryptCookieValue(t, rawState),
				}),
			)
			if tc.acceptHeader != "" {
				req.Header.Set(fiber.HeaderAccept, tc.acceptHeader)
			}

			resp, err := srv.Test(req)
			defer commontest.Close(t, resp)
			body := commontest.ToString(t, resp)

			require.NoError(t, err)
			require.Equal(t, tc.expectedStatus, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			assert.Contains(t, body, string(tc.expectedBodyPart))
			require.Len(t, resp.Cookies(), 2)
			assertCookie(t, expectedTokenCookie, resp.Cookies()[0])
			assertCookie(t, expectedSessionCookie, resp.Cookies()[1])
		})
	}
}

func TestExchangeInvalidInput(t *testing.T) {
	cases := []struct {
		name        string
		queryParams string
		cookieState *oidc.State
		svc         oidc.Service
		statusCode  int
	}{
		{
			name:        "No state cookie",
			queryParams: fmt.Sprintf("?code=myAuthCode&state=%s", encodeState(t, &oidc.State{ID: "state-0", Provider: "myProvider"})),
			cookieState: nil,
			svc:         &mockOIDCService{},
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "No auth code",
			queryParams: fmt.Sprintf("?state=%s", encodeState(t, &oidc.State{ID: "state-0", Provider: "myProvider"})),
			cookieState: &oidc.State{ID: "state-0", Provider: "myProvider"},
			svc:         &mockOIDCService{},
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "No auth state",
			queryParams: "?code=myAuthCode",
			cookieState: &oidc.State{ID: "state-0", Provider: "myProvider"},
			svc:         &mockOIDCService{},
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "No auth code and no auth state",
			cookieState: &oidc.State{ID: "state-0", Provider: "myProvider"},
			svc:         &mockOIDCService{},
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "State mismatch",
			queryParams: fmt.Sprintf("?code=myAuthCode&state=%s", encodeState(t, &oidc.State{ID: "state-1", Provider: "myProvider"})),
			cookieState: &oidc.State{ID: "state-0", Provider: "myProvider"},
			svc:         &mockOIDCService{},
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "Failed authentication",
			queryParams: fmt.Sprintf("?code=myAuthCode&state=%s", encodeState(t, &oidc.State{ID: "state-0", Provider: "myProvider"})),
			cookieState: &oidc.State{ID: "state-0", Provider: "myProvider"},
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
			srv := loginServer(tc.svc, staticTimeSvc)
			req := commontest.NewRequest(
				commontest.WithMethod(http.MethodGet),
				commontest.WithURL("http://localhost/login/callback"+tc.queryParams),
			)
			if tc.cookieState != nil {
				req.AddCookie(&http.Cookie{
					Name:  "TOKEN_STATE",
					Value: encryptCookieValue(t, encodeState(t, tc.cookieState)),
				})
			}

			resp, err := srv.Test(req)
			defer commontest.Close(t, resp)

			require.NoError(t, err)
			assertErrorResponse(t, resp, tc.statusCode)
		})
	}
}

func encodeState(t *testing.T, state *oidc.State) string {
	t.Helper()

	encodedState, err := state.Encode()
	require.NoError(t, err)

	return encodedState
}

func TestGetCurrentUser(t *testing.T) {
	svc := &mockOIDCService{
		FakeGetAuthenticatedUser: func(provider string, token *oidc.JSONWebToken) (*auth.User, error) {
			return &auth.User{
				Username: "myUser",
			}, nil
		},
	}
	srv := loginServer(svc, staticTimeSvc)
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
	assert.Equal(t, &commonhttp.ClientUser{Username: "myUser", Initials: "my"}, commontest.FromJSON[commonhttp.ClientUser](t, resp))
}

func TestGetCurrentUserNotLoggedIn(t *testing.T) {
	srv := loginServer(&mockOIDCService{}, staticTimeSvc)
	req := httptest.NewRequest(http.MethodGet, "http://localhost/user", nil)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assertErrorResponse(t, resp, http.StatusUnauthorized)
}

func TestLogout(t *testing.T) {
	svc := &mockOIDCService{
		FakeLogout: func(token *oidc.JSONWebToken) error {
			return nil
		},
	}
	srv := loginServer(svc, staticTimeSvc)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{Provider: "myProvider"})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/logout"),
		commontest.WithCookie(&http.Cookie{
			Name:  "SESSION",
			Value: encryptCookieValue(t, token),
		}),
	)
	expectedSessionCookie := &http.Cookie{
		Name:     "SESSION",
		Value:    "invalid",
		Expires:  expiresIn(-7 * 24 * time.Hour),
		SameSite: http.SameSiteStrictMode,
	}

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, resp.Cookies(), 1)
	assertCookie(t, expectedSessionCookie, resp.Cookies()[0])
}

func TestLogoutHTML(t *testing.T) {
	svc := &mockOIDCService{
		FakeLogout: func(token *oidc.JSONWebToken) error {
			return nil
		},
	}
	srv := loginServer(svc, staticTimeSvc)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{Provider: "myProvider"})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/logout"),
		commontest.WithCookie(&http.Cookie{
			Name:  "SESSION",
			Value: encryptCookieValue(t, token),
		}),
		commontest.WithAccept(fiber.MIMETextHTMLCharsetUTF8),
	)
	expectedSessionCookie := &http.Cookie{
		Name:     "SESSION",
		Value:    "invalid",
		Expires:  expiresIn(-7 * 24 * time.Hour),
		SameSite: http.SameSiteStrictMode,
	}

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	require.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, "/", resp.Header.Get(fiber.HeaderLocation))
	assert.Len(t, resp.Cookies(), 1)
	assertCookie(t, expectedSessionCookie, resp.Cookies()[0])
}

func TestLogoutNoSession(t *testing.T) {
	srv := loginServer(&mockOIDCService{}, staticTimeSvc)
	req := httptest.NewRequest(http.MethodGet, "http://localhost/logout", nil)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLogoutError(t *testing.T) {
	cases := []struct {
		name                 string
		sessionCookieValueFn func(t *testing.T) string
		statusCode           int
		src                  oidc.Service
	}{
		{
			name: "Invalid session value",
			sessionCookieValueFn: func(t *testing.T) string {
				return "123"
			},
			statusCode: http.StatusBadRequest,
			src:        &mockOIDCService{},
		},
		{
			name: "Failed logout",
			sessionCookieValueFn: func(t *testing.T) string {
				return commontest.Base64Encoded(t, &oidc.JSONWebToken{Provider: "myProvider"})
			},
			statusCode: http.StatusInternalServerError,
			src: &mockOIDCService{
				FakeLogout: func(token *oidc.JSONWebToken) error {
					return fmt.Errorf("some error")
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := loginServer(tc.src, staticTimeSvc)
			req := httptest.NewRequest(http.MethodGet, "http://localhost/logout", nil)
			value := tc.sessionCookieValueFn(t)
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

	require.Equal(t, expectedStatus, resp.StatusCode)
	assert.Equal(t, problemjson.ContentType, resp.Header.Get(fiber.HeaderContentType))

	result := commontest.FromJSON[problemjson.ProblemJSON](t, resp)
	assert.Equal(t, expectedStatus, result.Status)
}

func assertCookie(t *testing.T, expected *http.Cookie, actual *http.Cookie) {
	t.Helper()

	// these properties should always be set like this
	expected.HttpOnly = true
	expected.Path = "/"
	expected.Secure = true

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

func loginServer(svc oidc.Service, timeSvc common.TimeService) *server.Server {
	srv := server.NewHTTPTestServer()
	cookieEncryptionKey = srv.Cfg.Cookie.EncryptionKey

	cfgOidc := config.Oidc{
		StateCookieAge:    5 * time.Second,
		SessionCookieName: "SESSION",
		RedirectURI:       "http://localhost/home",
	}
	srv.RegisterRoutes(func(r fiber.Router) {
		loginadapter.Routes(r.Group("/"), cfgOidc, svc, timeSvc)
	})

	return srv
}
