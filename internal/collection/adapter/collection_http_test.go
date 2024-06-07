package adapter_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/collection/adapter"
	"github.com/konstantinfoerster/card-service-go/internal/collection/adapter/fakes"
	"github.com/konstantinfoerster/card-service-go/internal/collection/application"
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	oidcfakes "github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc/fakes"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
	"github.com/konstantinfoerster/card-service-go/internal/common/img"
	"github.com/konstantinfoerster/card-service-go/internal/common/server"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func collectServer(t *testing.T) *server.Server {
	srv := defaultServer()

	authSvc := oidcfakes.NewUserService(validUser)

	repo, err := fakes.NewRepository(img.NewPHasher())
	require.NoError(t, err)

	collectSvc := application.NewCollectionService(repo, repo)

	_, err = collectSvc.Collect(domain.Item{ID: 434, Amount: 1}, domain.Collector{ID: validUser.ID})
	require.NoError(t, err)
	_, err = collectSvc.Collect(domain.Item{ID: 514, Amount: 5}, domain.Collector{ID: validUser.ID})
	require.NoError(t, err)
	_, err = collectSvc.Collect(domain.Item{ID: 706, Amount: 3}, domain.Collector{ID: validUser.ID})
	require.NoError(t, err)

	srv.RegisterRoutes(func(r fiber.Router) {
		adapter.CollectRoutes(r.Group("/"), config.Oidc{}, authSvc, collectSvc)
	})

	return srv
}

func TestSearchCollected(t *testing.T) {
	srv := collectServer(t)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser.ID})
	cases := []struct {
		name                string
		header              func(req *http.Request)
		page                string
		expectedContentType string
		assertContent       func(t *testing.T, resp *http.Response)
	}{
		{
			name:                "default page as json",
			header:              func(req *http.Request) {},
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				expected := []adapter.Card{
					{Item: adapter.Item{ID: 434, Amount: 1}, Name: "Demonic Attorney"},
					{Item: adapter.Item{ID: 706, Amount: 3}, Name: "Demonic Hordes"},
					{Item: adapter.Item{ID: 514, Amount: 5}, Name: "Demonic Tutor"},
				}
				body := commontest.FromJSON[adapter.PagedResult[adapter.Card]](t, resp)
				assert.False(t, body.HasMore)
				assert.Equal(t, 1, body.Page)
				assert.ElementsMatch(t, expected, body.Data)
			},
		},
		{
			name: "default page as html",
			header: func(req *http.Request) {
				req.Header.Set(fiber.HeaderAccept, fiber.MIMETextHTMLCharsetUTF8)
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsFullHTML(t, body)
				commontest.AssertContainsProfile(t, body)
				assert.Contains(t, body, "data-testid=\"mycards-list\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name: "defaut page as htmx",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"mycards-list\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name:                "second page as json",
			header:              func(req *http.Request) {},
			page:                "size=1&page=2",
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				expected := []adapter.Card{
					{Item: adapter.Item{ID: 706, Amount: 3}, Name: "Demonic Hordes"},
				}
				body := commontest.FromJSON[adapter.PagedResult[adapter.Card]](t, resp)
				assert.True(t, body.HasMore)
				assert.Equal(t, 2, body.Page)
				assert.ElementsMatch(t, expected, body.Data)
			},
		},
		{
			name: "second page as htmx",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			page:                "size=1&page=2",
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsPartialHTML(t, body)
				assert.NotContains(t, body, "data-testid=\"mycards-list\"")
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-"), "expected 1 card in %s", body)
			},
		},
		{
			name: "htmx has lazy loading marker",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			page:                "size=1",
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				assert.Contains(t, body, "hidden-card")
			},
		},
		{
			name: "htmx has no lazy loading marker",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				assert.NotContains(t, body, "class=\"hidden-card\"")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := commontest.NewRequest(
				commontest.WithMethod(http.MethodGet),
				commontest.WithURL("http://localhost/mycards?name=Demonic&"+tc.page),
				commontest.WithEncryptedCookie(t, http.Cookie{
					Name:  "SESSION",
					Value: token,
				}),
			)
			tc.header(req)

			resp, err := srv.Test(req)
			defer commontest.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			tc.assertContent(t, resp)
		})
	}
}

func TestSearchCollectedNoSession(t *testing.T) {
	srv := collectServer(t)
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/mycards?name=Demonic"),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestCollectItemAdd(t *testing.T) {
	srv := collectServer(t)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser.ID})
	cases := []struct {
		name                string
		header              func(req *http.Request)
		amount              int
		expectedContentType string
		assertContent       func(t *testing.T, resp *http.Response)
	}{
		{
			name:                "add via rest api",
			header:              func(req *http.Request) {},
			amount:              1,
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.FromJSON[adapter.Item](t, resp)
				assert.Equal(t, &adapter.Item{ID: 1, Amount: 1}, body)
			},
		},
		{
			name: "add via htmx",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			amount:              1,
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsPartialHTML(t, body)

				assert.Contains(t, body, "data-testid=\"add-card-btn\"", "expect to have add button")
				assert.Contains(t, body, "data-testid=\"remove-card-btn\"", "expect remove button")
				assert.Contains(t, body, "hx-vals='{ \"id\": 1,\"amount\": 2 }'", "expect add value attributes")
				assert.Contains(t, body, "hx-vals='{ \"id\": 1,\"amount\": 0 }'", "expect remove value attributes")
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := commontest.NewRequest(
				commontest.WithMethod(http.MethodPost),
				commontest.WithURL("http://localhost/mycards"),
				commontest.WithEncryptedCookie(t, http.Cookie{
					Name:  "SESSION",
					Value: token,
				}),
				commontest.WithJSONBody(t, adapter.Item{ID: 1, Amount: tc.amount}),
			)
			tc.header(req)

			resp, err := srv.Test(req)
			defer commontest.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			tc.assertContent(t, resp)
		})
	}
}

func TestCollectItemRemove(t *testing.T) {
	srv := collectServer(t)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser.ID})
	cases := []struct {
		name                string
		header              func(req *http.Request)
		amount              int
		expectedContentType string
		assertContent       func(t *testing.T, resp *http.Response)
	}{
		{
			name: "remove via htmx",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsPartialHTML(t, body)

				assert.Contains(t, body, "data-testid=\"add-card-btn\"", "expect to have add button")
				assert.NotContains(t, body, "data-testid=\"remove-card-btn\"", "expect disabled remove button")
				assert.Contains(t, body, "hx-vals='{ \"id\": 1,\"amount\": 1 }'", "expect add value attributes")
				assert.Contains(t, body, "hx-vals='{ \"id\": 1,\"amount\": 0 }'", "expect remove value attributes")
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// collect item
			reqAdd := commontest.NewRequest(
				commontest.WithMethod(http.MethodPost),
				commontest.WithURL("http://localhost/mycards"),
				commontest.WithEncryptedCookie(t, http.Cookie{
					Name:  "SESSION",
					Value: token,
				}),
				commontest.HTMXRequest(),
				commontest.WithJSONBody(t, adapter.Item{ID: 1, Amount: 1}),
			)
			respAdd, _ := srv.Test(reqAdd)
			defer commontest.Close(t, respAdd)
			// remove collected item
			reqRemove := commontest.NewRequest(
				commontest.WithMethod(http.MethodPost),
				commontest.WithURL("http://localhost/mycards"),
				commontest.WithEncryptedCookie(t, http.Cookie{
					Name:  "SESSION",
					Value: token,
				}),
				commontest.WithJSONBody(t, adapter.Item{ID: 1, Amount: 0}),
			)
			tc.header(reqRemove)

			resp, err := srv.Test(reqRemove)
			defer commontest.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			tc.assertContent(t, resp)
		})
	}
}

func TestCollectItemWithInvalidUser(t *testing.T) {
	srv := collectServer(t)
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: invalidUser})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodPost),
		commontest.WithURL("http://localhost/mycards"),
		commontest.WithEncryptedCookie(t, http.Cookie{
			Name:  "SESSION",
			Value: token,
		}),
		commontest.WithJSONBody(t, adapter.Item{ID: 1}),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
