package cards_test

import (
	"context"
	"net/http"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/collection"
	"github.com/konstantinfoerster/card-service-go/internal/cards/detect"
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

func TestSearch(t *testing.T) {
	srv := searchServer(t)
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
					{ItemDTO: cards.ItemDTO{ID: 434}, Name: "Demonic Attorney"},
					{ItemDTO: cards.ItemDTO{ID: 706}, Name: "Demonic Hordes"},
					{ItemDTO: cards.ItemDTO{ID: 514}, Name: "Demonic Tutor"},
				}
				body := test.FromJSON[web.PagedResponse[cards.CardDTO]](t, resp)
				assert.False(t, body.HasMore)
				assert.Equal(t, 1, body.Page)
				assert.ElementsMatch(t, expected, body.Data)
			},
		},
		{
			name:                "second page as json",
			header:              func(req *http.Request) {},
			page:                "size=1&page=2",
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				expected := []cards.CardDTO{
					{ItemDTO: cards.ItemDTO{ID: 706}, Name: "Demonic Hordes"},
				}
				body := test.FromJSON[web.PagedResponse[cards.CardDTO]](t, resp)
				assert.True(t, body.HasMore)
				assert.Equal(t, 2, body.Page)
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
				test.AssertContainsLogin(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name: "default page as htmx",
			header: func(req *http.Request) {
				req.Header.Set(web.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := test.ToString(t, resp)
				test.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
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
				assert.NotContains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-"), "expected 1 card  in %s", body)
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
				assert.NotContains(t, body, "hidden-card")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := test.NewRequest(
				test.WithMethod(http.MethodGet),
				test.WithURL("http://localhost/cards?name=Demonic&"+tc.page),
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

func TestSearchWithUser(t *testing.T) {
	srv := searchServer(t)
	validUser := auth.User{ID: "myUser", Username: "myUser"}
	token := test.Base64Encoded(t, oidc.JSONWebToken{IDToken: validUser.ID})
	cases := []struct {
		name                string
		header              func(req *http.Request)
		expectedContentType string
		assertContent       func(t *testing.T, resp *http.Response)
	}{
		{
			name:                "default page as json",
			header:              func(req *http.Request) {},
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				expected := []cards.CardDTO{
					{ItemDTO: cards.ItemDTO{ID: 434}, Name: "Demonic Attorney"},
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
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
				assert.Equal(t, 3, strings.Count(body, "data-testid=\"add-card-btn\""))
				assert.Equal(t, 2, strings.Count(body, "data-testid=\"remove-card-btn\""))
			},
		},
		{
			name: "default page as htmx",
			header: func(req *http.Request) {
				req.Header.Set(web.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := test.ToString(t, resp)
				test.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
				assert.Equal(t, 3, strings.Count(body, "data-testid=\"add-card-btn\""))
				assert.Equal(t, 2, strings.Count(body, "data-testid=\"remove-card-btn\""))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := test.NewRequest(
				test.WithMethod(http.MethodGet),
				test.WithURL("http://localhost/cards?name=Demonic"),
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

func TestSearchWithInvalidUser(t *testing.T) {
	srv := searchServer(t)
	invalidUser := "invalidUser"
	token := test.Base64Encoded(t, &oidc.JSONWebToken{IDToken: invalidUser})
	req := test.NewRequest(
		test.WithMethod(http.MethodGet),
		test.WithURL("http://localhost/cards?name=Demonic&size=5&page=1"),
		test.WithEncryptedCookie(t, http.Cookie{
			Name:  "SESSION",
			Value: token,
		}),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func currentDir() string {
	_, cf, _, _ := runtime.Caller(0)

	return path.Join(path.Dir(cf))
}

func searchServer(t *testing.T) *web.Server {
	srv := web.NewHTTPTestServer()

	validUser := auth.User{ID: "myUser", Username: "myUser"}
	detectorUser := auth.User{ID: "detectorUser", Username: "detectorUser"}
	authSvc := oidcfakes.NewUserService(validUser, detectorUser)
	repo, err := fakes.NewRepository(detect.NewPHasher())
	require.NoError(t, err)
	collectSvc := collection.NewService(repo, repo)

	ctx := context.Background()

	item, err := collection.NewItem(514, 5, validUser.ID)
	require.NoError(t, err)
	_, err = collectSvc.Collect(ctx, item)
	require.NoError(t, err)

	item, err = collection.NewItem(706, 3, validUser.ID)
	require.NoError(t, err)
	_, err = collectSvc.Collect(ctx, item)
	require.NoError(t, err)
	
	item, err = collection.NewItem(1, 1, detectorUser.ID)
	require.NoError(t, err)
    _, err = collectSvc.Collect(ctx, item)
	require.NoError(t, err)

	searchSvc := cards.NewService(repo)
	srv.RegisterRoutes(func(r fiber.Router) {
		cards.Routes(r.Group("/"), config.Oidc{}, authSvc, searchSvc)
	})

	return srv
}
