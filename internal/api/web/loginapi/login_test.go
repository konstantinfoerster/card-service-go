package loginapi_test

import (
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	fibermemory "github.com/gofiber/storage/memory/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/api/web/loginapi"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var staticGenerator = auth.StaticGenerator{Value: "randVal"}
var authCfg = auth.Config{
	Session: auth.Cookie{
		Name:      "SESSION",
		ExpiresIn: 1 * time.Hour,
		SameSite:  fiber.CookieSameSiteStrictMode,
	},
	State: auth.Cookie{
		Name:      "STATE",
		ExpiresIn: 1 * time.Hour,
		SameSite:  fiber.CookieSameSiteLaxMode,
	},
}

func TestLogin(t *testing.T) {
	provider := auth.NewFakeProvider(
		auth.WithClaims(auth.NewClaim("myuser", "myUser")),
	)
	srv := loginServer(provider)
	req := test.NewRequest(
		test.WithMethod(web.MethodGet),
		test.WithURL("http://localhost/login/testProvider"),
	)
	genVal := staticGenerator.MustGenerateBase64Encoded()
	expectedState := encodedState(t)
	expectedStateCookie := &http.Cookie{
		Name:     authCfg.State.Name,
		Value:    expectedState,
		MaxAge:   int(authCfg.State.ExpiresIn.Seconds()),
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	}

	resp, err := srv.Test(req)
	defer test.Close(t, resp)
	require.NoError(t, err)

	location := resp.Header.Get("Location")
	require.Equal(t, web.StatusFound, resp.StatusCode)
	assert.Truef(t, strings.HasPrefix(location, "http://localhost/auth"), "location header want http://localhost/auth, got %s", location)
	assert.Contains(t, location, "state="+url.QueryEscape(genVal))
	assert.Contains(t, location, "code_challenge_method=S256")
	assert.Contains(t, location, "code_challenge="+url.QueryEscape(genVal))
	assert.Contains(t, location, "client_id=client-id")
	assert.Contains(t, location, "scope=openid")
	assert.Contains(t, location, "response_type=code")
	assert.Contains(t, location, "redirect_uri=http%3A%2F%2Flocalhost%2Fhome")
	assertEqualDecryptedCookie(t, expectedStateCookie, resp.Cookies()[0])
}

func TestLogin_UnknownProvider(t *testing.T) {
	srv := loginServer(nil)
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
	user := auth.NewClaim("myuser", "myUser")
	provider := auth.NewFakeProvider(
		auth.WithClaims(user),
	)
	srv := loginServer(provider)
	expectedSessionCookie := &http.Cookie{
		Name:     "SESSION",
		Value:    staticGenerator.Value,
		MaxAge:   int(authCfg.Session.ExpiresIn.Seconds()),
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	}
	expectedStateCookie := &http.Cookie{
		Name:     "STATE",
		Value:    "",
		MaxAge:   0,
		Expires:  time.Unix(0, 0).UTC(),
		SameSite: http.SameSiteLaxMode,
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
			rawState := staticGenerator.MustGenerateBase64Encoded()
			req := test.NewRequest(
				test.WithMethod(http.MethodGet),
				test.WithURL(fmt.Sprintf("http://localhost/login/testProvider/callback?code=%s&state=%s", user.UserID, rawState)),
				test.WithEncryptedCookie(t, "STATE", encodedState(t)),
			)
			if tc.acceptHeader != "" {
				req.Header.Set(fiber.HeaderAccept, tc.acceptHeader)
			}

			resp, err := srv.Test(req)
			defer test.Close(t, resp)
			body := test.ToString(t, resp.Body)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			assert.Contains(t, body, string(tc.expectedBodyPart))
			require.Len(t, resp.Cookies(), 2)
			assertEqualDecryptedCookie(t, expectedStateCookie, resp.Cookies()[0])
			assertEqualCookie(t, expectedSessionCookie, resp.Cookies()[1])
		})
	}
}

func TestExchange_InvalidInput(t *testing.T) {
	cases := []struct {
		name        string
		queryParams string
		provider    auth.Provider
		statusCode  int
	}{
		{
			name:        "unknown provider",
			queryParams: "",
			provider:    nil,
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "No state cookie",
			queryParams: fmt.Sprintf("?code=myUser&state=%s", staticGenerator.MustGenerateBase64Encoded()),
			provider:    nil,
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "No auth code",
			queryParams: fmt.Sprintf("?state=%s", staticGenerator.MustGenerateBase64Encoded()),
			provider:    auth.NewFakeProvider(),
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "No auth state",
			queryParams: "?code=myuser",
			provider:    auth.NewFakeProvider(auth.WithClaims(auth.NewClaim("myuser", "myUser"))),
			statusCode:  http.StatusBadRequest,
		},
		{
			name:       "No auth code and no auth state",
			provider:   auth.NewFakeProvider(),
			statusCode: http.StatusBadRequest,
		},
		{
			name:        "State mismatch",
			queryParams: fmt.Sprintf("?code=myuser&state=%s", base64.URLEncoding.EncodeToString([]byte("state-1"))),
			provider:    auth.NewFakeProvider(),
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "Authentication error",
			queryParams: fmt.Sprintf("?code=myAuthCode&state=%s", staticGenerator.MustGenerateBase64Encoded()),
			provider:    auth.NewFakeProvider(),
			statusCode:  http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := loginServer(tc.provider)
			opts := []test.RequestOpt{
				test.WithMethod(http.MethodGet),
				test.WithURL("http://localhost/login/testProvider/callback" + tc.queryParams),
			}
			if tc.provider != nil {
				opts = append(opts,
					test.WithEncryptedCookie(t, "STATE", encodedState(t)),
				)
			}
			req := test.NewRequest(opts...)

			resp, err := srv.Test(req)
			defer test.Close(t, resp)

			require.NoError(t, err)
			assertErrorResponse(t, resp, tc.statusCode)
		})
	}
}

func TestGetCurrentUser(t *testing.T) {
	claims := auth.NewClaim("myuser", "myUser")
	provider := auth.NewFakeProvider(auth.WithClaims(claims))
	srv := loginServer(provider)

	// finish login first
	rawState := staticGenerator.MustGenerateBase64Encoded()
	exchangeReq := test.NewRequest(
		test.WithMethod(http.MethodGet),
		test.WithURL(fmt.Sprintf("http://localhost/login/testProvider/callback?code=%s&state=%s", claims.UserID, rawState)),
		test.WithEncryptedCookie(t, "STATE", encodedState(t)),
	)
	exchangeResp, err := srv.Test(exchangeReq)
	require.NoError(t, err)
	test.Close(t, exchangeResp)

	req := test.NewRequest(
		test.WithMethod(http.MethodGet),
		test.WithURL("http://localhost/user"),
		test.WithSession(staticGenerator.Value),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, fiber.MIMEApplicationJSONCharsetUTF8, resp.Header.Get(fiber.HeaderContentType))
	assert.Equal(t, &web.ClientUser{Username: "myUser", Initials: "my"}, test.FromJSON[web.ClientUser](t, resp.Body))
}

func TestGetCurrentUser_UnencryptedSession(t *testing.T) {
	user := auth.NewClaim("myuser", "myUser")
	provider := auth.NewFakeProvider(auth.WithClaims(user))
	srv := loginServer(provider)
	req := test.NewRequest(
		test.WithMethod(http.MethodGet),
		test.WithURL("http://localhost/user"),
		test.WithSession(staticGenerator.Value),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assertErrorResponse(t, resp, http.StatusUnauthorized)
}

func TestGetCurrentUser_NotLoggedIn(t *testing.T) {
	srv := loginServer(nil)
	req := test.NewRequest(
		test.WithMethod(http.MethodGet),
		test.WithURL("http://localhost/user"),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assertErrorResponse(t, resp, http.StatusUnauthorized)
}

func TestLogout(t *testing.T) {
	cases := []struct {
		name             string
		opts             []test.RequestOpt
		login            bool
		expectedStatus   int
		expectedLocation string
		containsHeader   map[string]string
	}{
		{
			name: "json request with session",
			opts: []test.RequestOpt{
				test.WithAccept(fiber.MIMEApplicationJSONCharsetUTF8),
			},
			login:            true,
			expectedStatus:   http.StatusOK,
			expectedLocation: "",
		},
		{
			name: "html request with session",
			opts: []test.RequestOpt{
				test.WithAccept(fiber.MIMETextHTMLCharsetUTF8),
			},
			login:            true,
			expectedStatus:   http.StatusFound,
			expectedLocation: "/",
		},
		{
			name: "htmx request with session",
			opts: []test.RequestOpt{
				test.WithAccept(fiber.MIMETextHTMLCharsetUTF8),
				test.HTMXRequest(),
			},
			login:            true,
			expectedStatus:   http.StatusOK,
			expectedLocation: "",
			containsHeader: map[string]string{
				web.HeaderHTMXRefresh: "/",
			},
		},
		{
			name: "json request no session",
			opts: []test.RequestOpt{
				test.WithAccept(fiber.MIMEApplicationJSONCharsetUTF8),
			},
			login:            false,
			expectedStatus:   http.StatusOK,
			expectedLocation: "",
		},
		{
			name: "html request no session",
			opts: []test.RequestOpt{
				test.WithAccept(fiber.MIMETextHTMLCharsetUTF8),
			},
			login:            false,
			expectedStatus:   http.StatusFound,
			expectedLocation: "/",
		},
		{
			name: "htmx request no session",
			opts: []test.RequestOpt{
				test.WithAccept(fiber.MIMETextHTMLCharsetUTF8),
				test.HTMXRequest(),
			},
			login:            false,
			expectedStatus:   http.StatusOK,
			expectedLocation: "",
			containsHeader: map[string]string{
				web.HeaderHTMXRefresh: "/",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var srv *web.Server
			if tc.login {
				claims := auth.NewClaim("myuser", "myUser")
				provider := auth.NewFakeProvider(auth.WithClaims(claims))
				srv = loginServer(provider)
				// finish login first
				rawState := staticGenerator.MustGenerateBase64Encoded()
				exchangeReq := test.NewRequest(
					test.WithMethod(http.MethodGet),
					test.WithURL(fmt.Sprintf("http://localhost/login/testProvider/callback?code=%s&state=%s", claims.UserID, rawState)),
					test.WithEncryptedCookie(t, "STATE", encodedState(t)),
				)
				exchangeResp, err := srv.Test(exchangeReq)
				require.NoError(t, err)
				test.Close(t, exchangeResp)
				// session + state cookie
				require.Len(t, exchangeResp.Cookies(), 2)
			} else {
				srv = loginServer(nil)
			}

			// logout
			opts := slices.Clone(tc.opts)
			opts = append(opts,
				test.WithMethod(http.MethodPost),
				test.WithURL("http://localhost/logout"),
				test.WithSession(staticGenerator.Value),
			)
			req := test.NewRequest(opts...)
			expectedSessionCookie := &http.Cookie{
				Name:     "SESSION",
				Value:    "",
				MaxAge:   -1,
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
			// expired session cookie
			require.Len(t, resp.Cookies(), 1)
			assertEqualCookie(t, expectedSessionCookie, resp.Cookies()[0])
			for k, v := range tc.containsHeader {
				assert.Equal(t, resp.Header.Get(k), v)
			}
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

func assertEqualDecryptedCookie(t *testing.T, expected *http.Cookie, actual *http.Cookie) {
	t.Helper()

	actual.Value = test.DecryptCookieValue(t, actual.Value)
	actual.Raw = ""
	actual.RawExpires = ""

	assert.Equal(t, expected, actual)
}

func assertEqualCookie(t *testing.T, expected *http.Cookie, actual *http.Cookie) {
	t.Helper()

	actual.Raw = ""
	actual.RawExpires = ""

	assert.Equal(t, expected, actual)
}

func encodedState(t *testing.T) string {
	t.Helper()

	return test.Base64Encoded(t, auth.State{
		Value:    staticGenerator.MustGenerateBase64Encoded(),
		Verifier: staticGenerator.MustGenerateBase64Encoded(),
	})
}

func loginServer(provider auth.Provider) *web.Server {
	svc := auth.New(staticGenerator, auth.NewProviders(provider))
	srv := web.NewTestServer()

	gob.Register(auth.User{})
	sCfg := session.Config{
		KeyLookup:         "cookie:" + authCfg.Session.Name,
		Expiration:        authCfg.Session.ExpiresIn,
		CookieSecure:      true,
		CookieHTTPOnly:    true,
		CookieSessionOnly: false,
		CookieSameSite:    authCfg.Session.SameSite,
		Storage:           fibermemory.New(),
		KeyGenerator: func() string {
			return staticGenerator.Value
		},
	}
	store := session.New(sCfg)
	srv.RegisterRoutes(func(r fiber.Router) {
		loginapi.Routes(
			r.Group("/"),
			store,
			web.NewAuthMiddleware(store),
			authCfg,
			svc,
		)
	})

	return srv
}
