package loginapi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/api/web/loginapi"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/clock"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var staticTimeSvc = clock.NewFakeTimeService(time.Now())

var cookieEncryptionKey = ""

func TestLogin(t *testing.T) {
	validClaim := auth.NewClaims("myUser", "myUser")
	provider := auth.NewFakeProvider(
		auth.WithClaims(validClaim),
		auth.WithStateID("state-0"),
		auth.WithProviderCfg(auth.ProviderConfig{
			ClientID: "my-client-id",
			Scope:    "oidc email",
			AuthURL:  "http://localhost/auth",
		}),
	)
	srv := loginServer(staticTimeSvc, provider)
	req := test.NewRequest(
		test.WithMethod(web.MethodGet),
		test.WithURL("http://localhost/login/testProvider"),
	)
	expectedState := provider.GenerateState().Encode()
	expectedCookie := &http.Cookie{
		Name:     "TOKEN_STATE",
		Value:    expectedState,
		Expires:  expiresIn(5 * time.Second),
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	}

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	location := resp.Header.Get("Location")
	require.NoError(t, err)
	require.Equal(t, web.StatusFound, resp.StatusCode)
	assert.Truef(t, strings.HasPrefix(location, "http://localhost/auth"), "location header want http://localhost/auth, got %s", location)
	assert.Contains(t, location, "state="+expectedState)
	assert.Contains(t, location, "client_id=my-client-id")
	assert.Contains(t, location, "scope=oidc+email")
	assert.Contains(t, location, "response_type=code")
	assert.Contains(t, location, "redirect_uri=http%3A%2F%2Flocalhost%2Fhome")
	assertEqualCookie(t, expectedCookie, resp.Cookies()[0])
}

func TestLoginUnknownProvider(t *testing.T) {
	srv := loginServer(staticTimeSvc, nil)
	req := test.NewRequest(
		test.WithMethod(http.MethodGet),
		test.WithURL("http://localhost/login/unknownProvider"),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assertErrorResponse(t, resp, http.StatusBadRequest)
}

func TestExchangeCode(t *testing.T) {
	sessionExpires := staticTimeSvc.Now().Unix()
	validClaim := auth.NewClaims("myUser", "myUser")
	provider := auth.NewFakeProvider(
		auth.WithClaims(validClaim),
		auth.WithExpire(sessionExpires),
	)
	srv := loginServer(staticTimeSvc, provider)
	token := provider.Token(validClaim).Encode()
	expectedSessionCookie := &http.Cookie{
		Name:     "SESSION",
		Value:    token,
		Expires:  expiresIn(time.Duration(sessionExpires) * time.Second),
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	}
	expectedTokenCookie := &http.Cookie{
		Name:     "TOKEN_STATE",
		Value:    "invalid",
		Expires:  expiresIn(-7 * 24 * time.Hour),
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
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
			expectedBodyPart:    test.ToJSON(t, &web.ClientUser{Username: "myUser", Initials: "my"}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rawState := provider.GenerateState().Encode()
			req := test.NewRequest(
				test.WithMethod(http.MethodGet),
				test.WithURL(fmt.Sprintf("http://localhost/login/callback?code=myUser&state=%s", rawState)),
				test.WithCookie("TOKEN_STATE", encryptCookieValue(t, rawState)),
			)
			if tc.acceptHeader != "" {
				req.Header.Set(fiber.HeaderAccept, tc.acceptHeader)
			}

			resp, err := srv.Test(req)
			defer test.Close(t, resp)
			body := test.ToString(t, resp.Body)

			require.NoError(t, err)
			require.Equal(t, tc.expectedStatus, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			assert.Contains(t, body, string(tc.expectedBodyPart))
			require.Len(t, resp.Cookies(), 2)
			assertEqualCookie(t, expectedTokenCookie, resp.Cookies()[0])
			assertEqualCookie(t, expectedSessionCookie, resp.Cookies()[1])
		})
	}
}

func TestExchangeInvalidInput(t *testing.T) {
	validClaim := auth.NewClaims("myUser", "myUser")
	cases := []struct {
		name        string
		queryParams string
		provider    auth.Provider
		statusCode  int
	}{
		{
			name:        "No state cookie",
			queryParams: fmt.Sprintf("?code=myUser&state=%s", auth.State{ID: "state-0"}.Encode()),
			provider:    nil,
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "No auth code",
			queryParams: fmt.Sprintf("?state=%s", auth.State{ID: "state-0"}.Encode()),
			provider:    nil,
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "No auth state",
			queryParams: "?code=myUser",
			provider: auth.NewFakeProvider(
				auth.WithClaims(validClaim),
			),
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "No auth code and no auth state",
			provider:   auth.NewFakeProvider(),
			statusCode: http.StatusBadRequest,
		},
		{
			name: "State mismatch",
			queryParams: fmt.Sprintf("?code=myUser&state=%s",
				auth.State{ID: "state-1", Provider: "testProvider"}.Encode(),
			),
			provider:   auth.NewFakeProvider(),
			statusCode: http.StatusBadRequest,
		},
		{
			name: "Failed authentication",
			queryParams: fmt.Sprintf("?code=myAuthCode&state=%s",
				auth.State{ID: "state-0", Provider: "testProvider"}.Encode(),
			),
			provider:   auth.NewFakeProvider(),
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := loginServer(staticTimeSvc, tc.provider)
			req := test.NewRequest(
				test.WithMethod(http.MethodGet),
				test.WithURL("http://localhost/login/callback"+tc.queryParams),
			)
			if tc.provider != nil {
				state := tc.provider.GenerateState()
				req.AddCookie(&http.Cookie{
					Name:  "TOKEN_STATE",
					Value: encryptCookieValue(t, state.Encode()),
				})
			}

			resp, err := srv.Test(req)
			defer test.Close(t, resp)

			require.NoError(t, err)
			assertErrorResponse(t, resp, tc.statusCode)
		})
	}
}

func TestGetCurrentUser(t *testing.T) {
	validClaim := auth.NewClaims("myUser", "myUser")
	provider := auth.NewFakeProvider(auth.WithClaims(validClaim))
	srv := loginServer(staticTimeSvc, provider)
	token := provider.Token(validClaim).Encode()
	req := test.NewRequest(
		test.WithMethod(http.MethodGet),
		test.WithURL("http://localhost/user"),
		test.WithCookie("SESSION", encryptCookieValue(t, token)),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, fiber.MIMEApplicationJSONCharsetUTF8, resp.Header.Get(fiber.HeaderContentType))
	assert.Equal(t, &web.ClientUser{Username: "myUser", Initials: "my"}, test.FromJSON[web.ClientUser](t, resp.Body))
}

func TestAuthInfoNotLoggedIn(t *testing.T) {
	srv := loginServer(staticTimeSvc, nil)
	req := httptest.NewRequest(http.MethodGet, "http://localhost/user", nil)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assertErrorResponse(t, resp, http.StatusUnauthorized)
}

func TestLogout(t *testing.T) {
	cases := []struct {
		name             string
		acceptHeader     string
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:             "json response",
			acceptHeader:     fiber.MIMEApplicationJSONCharsetUTF8,
			expectedStatus:   http.StatusOK,
			expectedLocation: "",
		},
		{
			name:             "html response",
			acceptHeader:     fiber.MIMETextHTMLCharsetUTF8,
			expectedStatus:   http.StatusFound,
			expectedLocation: "/",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			validClaim := auth.NewClaims("myUser", "myUser")
			provider := auth.NewFakeProvider(auth.WithClaims(validClaim))

			srv := loginServer(staticTimeSvc, provider)
			token := provider.Token(validClaim).Encode()
			rawState := provider.GenerateState().Encode()
			reqLogin := test.NewRequest(
				test.WithMethod(http.MethodGet),
				test.WithURL(fmt.Sprintf("http://localhost/login/callback?code=myUser&state=%s", rawState)),
				test.WithCookie("TOKEN_STATE", encryptCookieValue(t, rawState)),
			)
			respLogin, err := srv.Test(reqLogin)
			require.NoError(t, err)
			defer test.Close(t, respLogin)

			req := test.NewRequest(
				test.WithMethod(http.MethodGet),
				test.WithURL("http://localhost/logout"),
				test.WithCookie("SESSION", encryptCookieValue(t, token)),
				test.WithAccept(tc.acceptHeader),
			)
			expectedSessionCookie := &http.Cookie{
				Name:     "SESSION",
				Value:    "invalid",
				Expires:  expiresIn(-7 * 24 * time.Hour),
				SameSite: http.SameSiteStrictMode,
				HttpOnly: true,
				Path:     "/",
				Secure:   true,
			}

			resp, err := srv.Test(req)
			defer test.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, tc.expectedStatus, resp.StatusCode)
			assert.Equal(t, tc.expectedLocation, resp.Header.Get(fiber.HeaderLocation))
			assert.Len(t, resp.Cookies(), 1)
			assertEqualCookie(t, expectedSessionCookie, resp.Cookies()[0])
		})
	}
}

func TestLogoutNoSession(t *testing.T) {
	srv := loginServer(staticTimeSvc, nil)
	req := httptest.NewRequest(http.MethodGet, "http://localhost/logout", nil)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLogoutError(t *testing.T) {
	cases := []struct {
		name                 string
		sessionCookieValueFn func(t *testing.T) string
		statusCode           int
	}{
		{
			name: "Invalid session value",
			sessionCookieValueFn: func(t *testing.T) string {
				return "123"
			},
			statusCode: http.StatusBadRequest,
		},
		{
			name: "Failed logout",
			sessionCookieValueFn: func(t *testing.T) string {
				invalidToken := &auth.JSONWebToken{Provider: "unknownProvider"}
				return invalidToken.Encode()
			},
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			validClaim := auth.NewClaims("myUser", "myUser")
			provider := auth.NewFakeProvider(auth.WithClaims(validClaim))
			srv := loginServer(staticTimeSvc, provider)
			req := httptest.NewRequest(http.MethodGet, "http://localhost/logout", nil)
			value := tc.sessionCookieValueFn(t)
			req.AddCookie(&http.Cookie{
				Name:  "SESSION",
				Value: encryptCookieValue(t, value),
			})

			resp, err := srv.Test(req)
			defer test.Close(t, resp)

			require.NoError(t, err)
			assertErrorResponse(t, resp, tc.statusCode)
		})
	}
}

func assertErrorResponse(t *testing.T, resp *http.Response, expectedStatus int) {
	t.Helper()

	require.Equal(t, expectedStatus, resp.StatusCode)
	assert.Equal(t, web.ContentType, resp.Header.Get(fiber.HeaderContentType))

	result := test.FromJSON[web.ProblemJSON](t, resp.Body)
	assert.Equal(t, expectedStatus, result.Status)
}

func assertEqualCookie(t *testing.T, expected *http.Cookie, actual *http.Cookie) {
	t.Helper()

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

func loginServer(timeSvc clock.TimeService, provider auth.Provider) *web.Server {
	oCfg := auth.OidcConfig{
		StateCookieAge:    5 * time.Second,
		SessionCookieName: "SESSION",
		RedirectURI:       "http://localhost/home",
	}
	svc := auth.New(oCfg, provider)
	srv := web.NewHTTPTestServer()
	cookieEncryptionKey = srv.Cfg.Cookie.EncryptionKey
	srv.RegisterRoutes(func(r fiber.Router) {
		loginapi.Routes(r.Group("/"), web.NewAuthMiddleware(oCfg, svc), oCfg, svc, timeSvc)
	})

	return srv
}