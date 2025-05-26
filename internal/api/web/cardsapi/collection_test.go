package cardsapi_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/api/web/cardsapi"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/cards/memory"
	"github.com/konstantinfoerster/card-service-go/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchCollected(t *testing.T) {
	srv, provider := testServer(t)
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
						ID:     "Y2FyZD0xMTQzNCZmYWNlPTExNDM0", // 11434
						Amount: 1,
						Name:   "Domonic Attorney",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
						Number: "104",
					},
					{
						ID:     "Y2FyZD0xMTcwNiZmYWNlPTExNzA2", // 11706
						Amount: 3,
						Name:   "Domonic Hordes",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
						Number: "105",
					},
					{
						ID:     "Y2FyZD0xMTUxNCZmYWNlPTExNTE0", // 11514
						Amount: 5,
						Name:   "Domonic Tutor",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
						Number: "106",
					},
				}
				body := test.FromJSON[cardsapi.PagedResponse[cardsapi.Card]](t, rBody)
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
				assert.Contains(t, body, "data-testid=\"mycards-list\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name: "defaut page as htmx",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"mycards-list\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name:                "second page as json",
			page:                "size=1&page=2",
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				expected := []cardsapi.Card{
					{
						ID:     "Y2FyZD0xMTcwNiZmYWNlPTExNzA2", // 11706
						Amount: 3,
						Number: "105",
						Name:   "Domonic Hordes",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
					},
				}
				body := test.FromJSON[cardsapi.PagedResponse[cardsapi.Card]](t, rBody)
				assert.True(t, body.HasMore)
				assert.Equal(t, 2, body.Page)
				assert.ElementsMatch(t, expected, body.Data)
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
				assert.NotContains(t, body, "data-testid=\"mycards-list\"")
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-"), "expected 1 card in %s", body)
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
				assert.NotContains(t, body, "class=\"hidden-card\"")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := provider.Token("myuser")
			req := test.NewRequest(
				test.WithMethod(web.MethodGet),
				test.WithURL("http://localhost/mycards?name=Domonic&"+tc.page),
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

func TestSearchCollectedNoSession(t *testing.T) {
	srv, _ := testServer(t)
	req := test.NewRequest(
		test.WithMethod(web.MethodGet),
		test.WithURL("http://localhost/mycards?name=Demonic"),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, web.StatusUnauthorized, resp.StatusCode)
}

func TestCollectItemAdd(t *testing.T) {
	srv, provider := testServer(t)
	cases := []struct {
		name                string
		header              map[string]string
		amount              int
		expectedContentType string
		assertContent       func(t *testing.T, rBody io.Reader)
	}{
		{
			name:                "add via rest api",
			amount:              1,
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.FromJSON[cardsapi.Item](t, rBody)
				assert.Equal(t, &cardsapi.Item{ID: "Y2FyZD0xMjQwNg==", Amount: 1}, body)
			},
		},
		{
			name: "add via htmx",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			amount:              1,
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)

				assert.Contains(t, body, "data-testid=\"add-card-btn\"", "expect to have add button")
				assert.Contains(t, body, "data-testid=\"remove-card-btn\"", "expect remove button")
				assert.Contains(t, body, "hx-vals='{ \"id\": \"Y2FyZD0xMjQwNg==\",\"amount\": 2 }'", "expect add value attributes")
				assert.Contains(t, body, "hx-vals='{ \"id\": \"Y2FyZD0xMjQwNg==\",\"amount\": 0 }'", "expect remove value attributes")
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := provider.Token("myuser")
			req := test.NewRequest(
				test.WithMethod(web.MethodPost),
				test.WithURL("http://localhost/mycards"),
				test.WithEncryptedCookie(t, "SESSION", test.Base64Encoded(t, token)),
				test.WithHeader(tc.header),
				test.WithJSONBody(t, cardsapi.NewItem(cards.NewID(12406), tc.amount)),
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

func TestCollectItemRemove(t *testing.T) {
	srv, provider := testServer(t)
	cases := []struct {
		name                string
		header              map[string]string
		amount              int
		expectedContentType string
		assertContent       func(t *testing.T, rBody io.Reader)
	}{
		{
			name: "remove via htmx",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)

				assert.Contains(t, body, "data-testid=\"add-card-btn\"", "expect to have add button")
				assert.NotContains(t, body, "data-testid=\"remove-card-btn\"", "expect disabled remove button")
				assert.Contains(t, body, "hx-vals='{ \"id\": \"Y2FyZD0xMjQwNg==\",\"amount\": 1 }'", "expect add value attributes")
				assert.Contains(t, body, "hx-vals='{ \"id\": \"Y2FyZD0xMjQwNg==\",\"amount\": 0 }'", "expect remove value attributes")
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := provider.Token("myuser")
			// collect item
			reqAdd := test.NewRequest(
				test.WithMethod(web.MethodPost),
				test.WithURL("http://localhost/mycards"),
				test.WithEncryptedCookie(t, "SESSION", test.Base64Encoded(t, token)),
				test.WithJSONBody(t, cardsapi.Item{ID: "Y2FyZD0xMjQwNg==", Amount: 1}),
			)
			respAdd, _ := srv.Test(reqAdd)
			defer test.Close(t, respAdd)
			// remove collected item
			reqRemove := test.NewRequest(
				test.WithMethod(web.MethodPost),
				test.WithURL("http://localhost/mycards"),
				test.WithEncryptedCookie(t, "SESSION", test.Base64Encoded(t, token)),
				test.WithHeader(tc.header),
				test.WithJSONBody(t, cardsapi.Item{ID: "Y2FyZD0xMjQwNg==", Amount: 0}),
			)

			resp, err := srv.Test(reqRemove)
			defer test.Close(t, resp)

			require.NoError(t, err)
			require.Equal(t, web.StatusOK, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get(fiber.HeaderContentType))
			tc.assertContent(t, resp.Body)
		})
	}
}

func TestCollectItemNoSession(t *testing.T) {
	srv, _ := testServer(t)
	req := test.NewRequest(
		test.WithMethod(web.MethodPost),
		test.WithURL("http://localhost/mycards"),
		test.WithJSONBody(t, cardsapi.Item{ID: "Y2FyZD0x"}),
	)

	resp, err := srv.Test(req)
	defer test.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, web.StatusUnauthorized, resp.StatusCode)
}

func testServer(t *testing.T) (*web.Server, *auth.FakeProvider) {
	seed, err := test.CardSeed()
	require.NoError(t, err)
	repo, err := memory.NewCollectRepository(seed)
	require.NoError(t, err)

	collectSvc := cards.NewCollectionService(repo)

	validClaim := auth.NewClaims("myuser", "myUser")
	collector := cards.NewCollector(validClaim.ID)
	ctx := context.Background()
	_, err = collectSvc.Collect(ctx, cards.Collectable{ID: cards.NewID(11434), Amount: 1}, collector)
	require.NoError(t, err)
	_, err = collectSvc.Collect(ctx, cards.Collectable{ID: cards.NewID(11514), Amount: 5}, collector)
	require.NoError(t, err)
	_, err = collectSvc.Collect(ctx, cards.Collectable{ID: cards.NewID(11706), Amount: 3}, collector)
	require.NoError(t, err)

	oCfg := auth.Config{}
	provider := auth.NewFakeProvider(auth.WithClaims(validClaim))
	authSvc := auth.New(oCfg, auth.NewProviders(provider))
	srv := web.NewTestServer()
	srv.RegisterRoutes(func(r fiber.Router) {
		cardsapi.CollectionRoutes(r.Group("/"), web.NewAuthMiddleware(oCfg, authSvc), collectSvc)
	})

	return srv, provider
}
