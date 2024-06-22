package collection_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/collection"
	"github.com/konstantinfoerster/card-service-go/internal/cards/fakes"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	oidcfakes "github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc/fakes"
	"github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/konstantinfoerster/card-service-go/internal/common/web"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validUser = auth.User{ID: "myUser", Username: "myUser"}

const invalidUser = "invalidUser"

func TestSearchCollected(t *testing.T) {
	srv := testServer(t)
	token := test.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser.ID})
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
				expected := []cards.CardDTO{
					{ItemDTO: cards.ItemDTO{ID: 434, Amount: 1}, Name: "Demonic Attorney"},
					{ItemDTO: cards.ItemDTO{ID: 706, Amount: 3}, Name: "Demonic Hordes"},
					{ItemDTO: cards.ItemDTO{ID: 514, Amount: 5}, Name: "Demonic Tutor"},
				}
				body := test.FromJSON[web.PagedResponse[cards.CardDTO]](t, resp)
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
				body := test.ToString(t, resp)
				test.AssertContainsFullHTML(t, body)
				test.AssertContainsProfile(t, body)
				assert.Contains(t, body, "data-testid=\"mycards-list\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name: "defaut page as htmx",
			header: func(req *http.Request) {
				req.Header.Set(web.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := test.ToString(t, resp)
				test.AssertContainsPartialHTML(t, body)
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
				expected := []cards.CardDTO{
					{ItemDTO: cards.ItemDTO{ID: 706, Amount: 3}, Name: "Demonic Hordes"},
				}
				body := test.FromJSON[web.PagedResponse[cards.CardDTO]](t, resp)
				assert.True(t, body.HasMore)
				assert.Equal(t, 2, body.Page)
				assert.ElementsMatch(t, expected, body.Data)
			},
		},
		{
			name: "second page as htmx",
			header: func(req *http.Request) {
				req.Header.Set(web.HeaderHTMXRequest, "true")
			},
			page:                "size=1&page=2",
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := test.ToString(t, resp)
				test.AssertContainsPartialHTML(t, body)
				assert.NotContains(t, body, "data-testid=\"mycards-list\"")
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-"), "expected 1 card in %s", body)
			},
		},
		{
			name: "htmx has lazy loading marker",
			header: func(req *http.Request) {
				req.Header.Set(web.HeaderHTMXRequest, "true")
			},
			page:                "size=1",
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := test.ToString(t, resp)
				assert.Contains(t, body, "hidden-card")
			},
		},
		{
			name: "htmx has no lazy loading marker",
			header: func(req *http.Request) {
				req.Header.Set(web.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := test.ToString(t, resp)
				assert.NotContains(t, body, "class=\"hidden-card\"")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := test.NewRequest(
				test.WithMethod(http.MethodGet),
				test.WithURL("http://localhost/mycards?name=Demonic&"+tc.page),
				test.WithEncryptedCookie(t, http.Cookie{
					Name:  "SESSION",
					Value: token,
				}),
			)
			tc.header(req)

			resp, err := srv.Test(req)
			defer test.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			tc.assertContent(t, resp)
		})
	}
}

func TestSearchCollectedNoSession(t *testing.T) {
	srv := testServer(t)
	req := test.NewRequest(
		test.WithMethod(http.MethodGet),
		test.WithURL("http://localhost/mycards?name=Demonic"),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestCollectItemAdd(t *testing.T) {
	srv := testServer(t)
	token := test.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser.ID})
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
				body := test.FromJSON[cards.ItemDTO](t, resp)
				assert.Equal(t, &cards.ItemDTO{ID: 1, Amount: 1}, body)
			},
		},
		{
			name: "add via htmx",
			header: func(req *http.Request) {
				req.Header.Set(web.HeaderHTMXRequest, "true")
			},
			amount:              1,
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := test.ToString(t, resp)
				test.AssertContainsPartialHTML(t, body)

				assert.Contains(t, body, "data-testid=\"add-card-btn\"", "expect to have add button")
				assert.Contains(t, body, "data-testid=\"remove-card-btn\"", "expect remove button")
				assert.Contains(t, body, "hx-vals='{ \"id\": 1,\"amount\": 2 }'", "expect add value attributes")
				assert.Contains(t, body, "hx-vals='{ \"id\": 1,\"amount\": 0 }'", "expect remove value attributes")
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := test.NewRequest(
				test.WithMethod(http.MethodPost),
				test.WithURL("http://localhost/mycards"),
				test.WithEncryptedCookie(t, http.Cookie{
					Name:  "SESSION",
					Value: token,
				}),
				test.WithJSONBody(t, cards.ItemDTO{ID: 1, Amount: tc.amount}),
			)
			tc.header(req)

			resp, err := srv.Test(req)
			defer test.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			tc.assertContent(t, resp)
		})
	}
}

func TestCollectItemRemove(t *testing.T) {
	srv := testServer(t)
	token := test.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser.ID})
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
				req.Header.Set(web.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := test.ToString(t, resp)
				test.AssertContainsPartialHTML(t, body)

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
			reqAdd := test.NewRequest(
				test.WithMethod(http.MethodPost),
				test.WithURL("http://localhost/mycards"),
				test.WithEncryptedCookie(t, http.Cookie{
					Name:  "SESSION",
					Value: token,
				}),
				test.HTMXRequest(),
				test.WithJSONBody(t, cards.ItemDTO{ID: 1, Amount: 1}),
			)
			respAdd, _ := srv.Test(reqAdd)
			defer test.Close(t, respAdd)
			// remove collected item
			reqRemove := test.NewRequest(
				test.WithMethod(http.MethodPost),
				test.WithURL("http://localhost/mycards"),
				test.WithEncryptedCookie(t, http.Cookie{
					Name:  "SESSION",
					Value: token,
				}),
				test.WithJSONBody(t, cards.ItemDTO{ID: 1, Amount: 0}),
			)
			tc.header(reqRemove)

			resp, err := srv.Test(reqRemove)
			defer test.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			tc.assertContent(t, resp)
		})
	}
}

func TestCollectItemWithInvalidUser(t *testing.T) {
	srv := testServer(t)
	token := test.Base64Encoded(t, &oidc.JSONWebToken{IDToken: invalidUser})
	req := test.NewRequest(
		test.WithMethod(http.MethodPost),
		test.WithURL("http://localhost/mycards"),
		test.WithEncryptedCookie(t, http.Cookie{
			Name:  "SESSION",
			Value: token,
		}),
		test.WithJSONBody(t, cards.ItemDTO{ID: 1}),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func testServer(t *testing.T) *web.Server {
	srv := web.NewHTTPTestServer()

	authSvc := oidcfakes.NewUserService(validUser)
	repo, err := fakes.NewRepository(nil)
	require.NoError(t, err)
	collectSvc := collection.NewService(repo)

	ctx := context.Background()
	collector := cards.NewCollector(validUser.ID)
	_, err = collectSvc.Collect(ctx, collection.Item{ID: 434, Amount: 1}, collector)
	require.NoError(t, err)
	_, err = collectSvc.Collect(ctx, collection.Item{ID: 514, Amount: 5}, collector)
	require.NoError(t, err)
	_, err = collectSvc.Collect(ctx, collection.Item{ID: 706, Amount: 3}, collector)
	require.NoError(t, err)

	srv.RegisterRoutes(func(r fiber.Router) {
		collection.Routes(r.Group("/"), config.Oidc{}, authSvc, collectSvc)
	})

	return srv
}
