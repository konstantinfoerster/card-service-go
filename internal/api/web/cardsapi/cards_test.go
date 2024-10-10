package cardsapi_test

import (
	"io"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/api/web/cardsapi"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/memory"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	srv, _ := searchServer(t)
	cases := []struct {
		name                string
		header              map[string]string
		page                string
		expectedContentType string
		assertContent       func(t *testing.T, rBody io.Reader)
	}{
		{
			name:                "default page as json",
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				expected := []cardsapi.Card{
					{
						ID:   434,
						Name: "Demonic Attorney",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
					},
					{
						ID:   706,
						Name: "Demonic Hordes",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
					},
					{
						ID:   514,
						Name: "Demonic Tutor",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
					},
				}
				body := test.FromJSON[cardsapi.PagedResponse](t, rBody)
				assert.False(t, body.HasMore)
				assert.Equal(t, 1, body.Page)
				assert.ElementsMatch(t, expected, body.Data)
			},
		},
		{
			name:                "second page as json",
			page:                "size=1&page=2",
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				expected := []cardsapi.Card{
					{
						ID:   706,
						Name: "Demonic Hordes",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
					},
				}
				body := test.FromJSON[cardsapi.PagedResponse](t, rBody)
				assert.True(t, body.HasMore)
				assert.Equal(t, 2, body.Page)
				assert.ElementsMatch(t, expected, body.Data)
			},
		},
		{
			name: "default page as html",
			header: map[string]string{
				fiber.HeaderAccept: fiber.MIMETextHTMLCharsetUTF8,
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsFullHTML(t, body)
				test.AssertContainsLogin(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name: "default page as htmx",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name: "second page as htmx",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			page:                "size=1&page=2",
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.NotContains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-"), "expected 1 card  in %s", body)
			},
		},
		{
			name: "htmx has lazy loading marker",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			page:                "size=1",
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				assert.Contains(t, body, "hidden-card")
			},
		},
		{
			name: "htmx has no lazy loading marker",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				assert.NotContains(t, body, "hidden-card")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := test.NewRequest(
				test.WithMethod(web.MethodGet),
				test.WithURL("http://localhost/cards?name=Demonic&"+tc.page),
				test.WithHeader(tc.header),
			)

			resp, err := srv.Test(req)
			defer test.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, web.StatusOK, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			tc.assertContent(t, resp.Body)
		})
	}
}

func TestSearchWithUser(t *testing.T) {
	srv, provider := searchServer(t)
	cases := []struct {
		name                string
		header              map[string]string
		expectedContentType string
		assertContent       func(t *testing.T, rBoy io.Reader)
	}{
		{
			name:                "default page as json",
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				expected := []cardsapi.Card{
					{
						ID:   434,
						Name: "Demonic Attorney",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
					},
					{
						ID:     706,
						Amount: 3,
						Name:   "Demonic Hordes",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
					},
					{
						ID:     514,
						Amount: 5,
						Name:   "Demonic Tutor",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
					},
				}
				body := test.FromJSON[cardsapi.PagedResponse](t, rBody)
				assert.False(t, body.HasMore)
				assert.Equal(t, 1, body.Page)
				assert.ElementsMatch(t, expected, body.Data)
			},
		},
		{
			name: "default page as html",
			header: map[string]string{
				fiber.HeaderAccept: fiber.MIMETextHTMLCharsetUTF8,
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
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
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
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
			token := provider.Token("myuser")
			req := test.NewRequest(
				test.WithMethod(web.MethodGet),
				test.WithURL("http://localhost/cards?name=Demonic"),
				test.WithEncryptedCookie(t, "SESSION", test.Base64Encoded(t, token)),
				test.WithHeader(tc.header),
			)

			resp, err := srv.Test(req)
			defer test.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, web.StatusOK, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			tc.assertContent(t, resp.Body)
		})
	}
}

func TestSearchWithInvalidUser(t *testing.T) {
	srv, provider := searchServer(t)
	token := &auth.JWT{
		Provider:    provider.GetName(),
		AccessToken: "invalidToken",
	}
	req := test.NewRequest(
		test.WithMethod(web.MethodGet),
		test.WithURL("http://localhost/cards?name=Demonic&size=5&page=1"),
		test.WithEncryptedCookie(t, "SESSION", test.Base64Encoded(t, token)),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, web.StatusUnauthorized, resp.StatusCode)
}

func searchServer(t *testing.T) (*web.Server, *auth.FakeProvider) {
	srv := web.NewHTTPTestServer()

	seed, err := test.CardSeed()
	require.NoError(t, err)

	validClaim := auth.NewClaims("myuser", "myUser")
	item1, err := cards.NewCollectable(514, 5)
	require.NoError(t, err)
	item2, err := cards.NewCollectable(706, 3)
	require.NoError(t, err)
	item3, err := cards.NewCollectable(1, 1)
	require.NoError(t, err)
	collected := map[string][]cards.Collectable{
		validClaim.ID: {item1, item2, item3},
	}

	repo, err := memory.NewCardRepository(seed, collected)
	require.NoError(t, err)

	oCfg := config.Oidc{}
	provider := auth.NewFakeProvider(auth.WithClaims(validClaim))
	authSvc := auth.New(oCfg, auth.NewProviders(provider))
	searchSvc := cards.NewCardService(repo)
	srv.RegisterRoutes(func(r fiber.Router) {
		cardsapi.SearchRoutes(r.Group("/"), web.NewAuthMiddleware(oCfg, authSvc), searchSvc)
	})

	return srv, provider
}
