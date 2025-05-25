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
						ID:   "Y2FyZD00MzQmZmFjZT00MzQ=", // 434
						Name: "Demonic Attorney",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
						Number: "103",
					},
					{
						ID:   "Y2FyZD03MDYmZmFjZT03MDY=", // 706
						Name: "Demonic Hordes",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
						Number: "104",
					},
					{
						ID:   "Y2FyZD01MTQmZmFjZT01MTQ=", // 514
						Name: "Demonic Tutor",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
						Number: "105",
					},
				}
				body := test.FromJSON[cardsapi.PagedResponse[cardsapi.Card]](t, rBody)
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
						ID:   "Y2FyZD03MDYmZmFjZT03MDY=", // 706
						Name: "Demonic Hordes",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
						Number: "104",
					},
				}
				body := test.FromJSON[cardsapi.PagedResponse[cardsapi.Card]](t, rBody)
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
						ID:   "Y2FyZD00MzQmZmFjZT00MzQ=", // 434
						Name: "Demonic Attorney",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
						Number: "103",
					},
					{
						ID:     "Y2FyZD03MDYmZmFjZT03MDY=", // 706
						Amount: 3,
						Name:   "Demonic Hordes",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
						Number: "104",
					},
					{
						ID:     "Y2FyZD01MTQmZmFjZT01MTQ=", // 514
						Amount: 5,
						Name:   "Demonic Tutor",
						Set: cardsapi.Set{
							Name: "Unlimited Edition",
							Code: "2ED",
						},
						Number: "105",
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

func TestDetail(t *testing.T) {
	srv, provider := searchServer(t)
	token := provider.Token("myuser")
	cases := []struct {
		name                string
		header              map[string]string
		cardID              string
		user                func() test.RequestOpt
		expectedContentType string
		assertContent       func(t *testing.T, rBody io.Reader)
	}{
		{
			name: "card detail with few prints has no more button",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			cardID:              "Y2FyZD0zMyZmYWNlPTMz", // 33
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"card-detail\"")
				assert.Contains(t, body, "data-testid=\"card-detail-img\"")
				assert.NotContains(t, body, "data-testid=\"card-print-more\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-print-id"), "expected 3 prints in %s", body)
				assert.Equalf(t, 2, strings.Count(body, "data-testid=\"card-detail-link"), "expected 2 card detail links in %s", body)
			},
		},
		{
			name: "card detail with many prints has more button",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			cardID:              "Y2FyZD00MDYmZmFjZT00MDY=", // 406
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"card-detail\"")
				assert.Contains(t, body, "data-testid=\"card-detail-img\"")
				assert.Contains(t, body, "data-testid=\"card-print-more\"")
				assert.Equalf(t, 10, strings.Count(body, "data-testid=\"card-print-id"), "expected 10 prints in %s", body)
				assert.Equalf(t, 9, strings.Count(body, "data-testid=\"card-detail-link"), "expected 9 card detail links in %s", body)
			},
		},
		{
			name: "card detail with multiple collected prints shows correct amount",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			user: func() test.RequestOpt {
				return test.WithEncryptedCookie(t, "SESSION", test.Base64Encoded(t, token))
			},
			cardID:              "Y2FyZD01ODImZmFjZT01ODI=", // 582
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"card-detail\"")
				assert.Contains(t, body, "data-testid=\"card-detail-img\"")
				assert.NotContains(t, body, "data-testid=\"card-print-more\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-print-id-"), "expected 3 prints in %s", body)
				assert.Equalf(t, 2, strings.Count(body, "data-testid=\"card-detail-link"), "expected 2 card detail links in %s", body)
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-print-amount-03"), "expected 1 print with amount 03 in %s", body)
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-print-amount-01"), "expected 1 print with amount 01 in %s", body)
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-print-amount-00"), "expected 1 print with amount 00 in %s", body)
				assert.Equal(t, 1, strings.Count(body, "data-testid=\"add-card-btn\""))
				assert.Equal(t, 1, strings.Count(body, "data-testid=\"remove-card-btn\""))
			},
		},
		{
			name: "card detail with user and uncollected card",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			user: func() test.RequestOpt {
				return test.WithEncryptedCookie(t, "SESSION", test.Base64Encoded(t, token))
			},
			cardID:              "Y2FyZD00MzQmZmFjZT00MzQ=", // 434
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"card-detail\"")
				assert.Contains(t, body, "data-testid=\"card-detail-img\"")
				assert.NotContains(t, body, "data-testid=\"card-print-more\"")
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-print-id-"), "expected 1 prints in %s", body)
				assert.Equalf(t, 0, strings.Count(body, "data-testid=\"card-detail-link"), "expected 0 card detail links in %s", body)
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-print-amount-00"), "expected 1 print with amount 00 in %s", body)
				assert.Equal(t, 1, strings.Count(body, "data-testid=\"add-card-btn\""))
				assert.Equal(t, 0, strings.Count(body, "data-testid=\"remove-card-btn\""), "expected no remove button")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user := tc.user
			if user == nil {
				user = func() test.RequestOpt { return nil }
			}

			req := test.NewRequest(
				test.WithMethod(web.MethodGet),
				test.WithURLf("http://localhost/cards/%s", tc.cardID),
				test.WithHeader(tc.header),
				user(),
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

func TestPrints(t *testing.T) {
	srv, provider := searchServer(t)
	token := provider.Token("myuser")
	cases := []struct {
		name                string
		header              map[string]string
		cardID              string
		queryParameter      string
		user                func() test.RequestOpt
		expectedContentType string
		assertContent       func(t *testing.T, rBody io.Reader)
	}{
		{
			name: "card prints with few prints has no more button",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			cardID:              "Y2FyZD0zMyZmYWNlPTMz", // 33
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.NotContains(t, body, "data-testid=\"card-print-more\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-print-id"), "expected 3 prints in %s", body)
				assert.Equalf(t, 2, strings.Count(body, "data-testid=\"card-detail-link"), "expected 2 card detail links in %s", body)
			},
		},
		{
			name: "card prints without user does not render amount",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			cardID:              "Y2FyZD0zMyZmYWNlPTMz", // 33
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.Equalf(t, 0, strings.Count(body, "data-testid=\"card-print-amount"), "expected no amount rendered in %s", body)
			},
		},
		{
			name: "card prints with size parameter",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			queryParameter:      "?size=1",
			cardID:              "Y2FyZD0zMyZmYWNlPTMz", // 33
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"card-print-more\"")
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-print-id"), "expected 1 prints in %s", body)
				assert.Equalf(t, 0, strings.Count(body, "data-testid=\"card-detail-link"), "expected 0 card detail links in %s", body)
			},
		},
		{
			name: "card prints many prints has more button",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			cardID:              "Y2FyZD00MDYmZmFjZT00MDY=", // 406
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"card-print-more\"")
				assert.Equalf(t, 10, strings.Count(body, "data-testid=\"card-print-id"), "expected 10 prints in %s", body)
				assert.Equalf(t, 9, strings.Count(body, "data-testid=\"card-detail-link"), "expected 9 card detail links in %s", body)
			},
		},
		{
			name: "card prints second page with last card",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			cardID:              "Y2FyZD00MDYmZmFjZT00MDY=", // 406
			queryParameter:      "?page=2",
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.NotContains(t, body, "data-testid=\"card-print-more\"")
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-print-id"), "expected 1 prints in %s", body)
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-detail-link"), "expected 1 card detail links in %s", body)
			},
		},
		{
			name: "card prints with multiple collected prints shows correct amount",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			user: func() test.RequestOpt {
				return test.WithEncryptedCookie(t, "SESSION", test.Base64Encoded(t, token))
			},
			cardID:              "Y2FyZD01ODImZmFjZT01ODI=", // 582
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.NotContains(t, body, "data-testid=\"card-print-more\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-print-id-"), "expected 3 prints in %s", body)
				assert.Equalf(t, 2, strings.Count(body, "data-testid=\"card-detail-link"), "expected 2 card detail links in %s", body)
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-print-amount-03"), "expected 1 print with amount 03 in %s", body)
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-print-amount-01"), "expected 1 print with amount 01 in %s", body)
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-print-amount-00"), "expected 1 print with amount 00 in %s", body)
			},
		},
		{
			name: "card prints with user and uncollected card",
			header: map[string]string{
				web.HeaderHTMXRequest: "true",
			},
			user: func() test.RequestOpt {
				return test.WithEncryptedCookie(t, "SESSION", test.Base64Encoded(t, token))
			},
			cardID:              "Y2FyZD00MzQmZmFjZT00MzQ=", // 434
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, rBody io.Reader) {
				body := test.ToString(t, rBody)
				test.AssertContainsPartialHTML(t, body)
				assert.NotContains(t, body, "data-testid=\"card-print-more\"")
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-print-id-"), "expected 1 prints in %s", body)
				assert.Equalf(t, 0, strings.Count(body, "data-testid=\"card-detail-link"), "expected 0 card detail links in %s", body)
				assert.Equalf(t, 1, strings.Count(body, "data-testid=\"card-print-amount-00"), "expected 1 print with amount 00 in %s", body)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user := tc.user
			if user == nil {
				user = func() test.RequestOpt { return nil }
			}

			req := test.NewRequest(
				test.WithMethod(web.MethodGet),
				test.WithURLf("http://localhost/cards/%s/prints%s", tc.cardID, tc.queryParameter),
				test.WithHeader(tc.header),
				user(),
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
func searchServer(t *testing.T) (*web.Server, *auth.FakeProvider) {
	srv := web.NewTestServer()

	seed, err := test.CardSeed()
	require.NoError(t, err)

	validClaim := auth.NewClaims("myuser", "myUser")
	item1, err := cards.NewCollectable(cards.NewID(514), 5)
	require.NoError(t, err)
	item2, err := cards.NewCollectable(cards.NewID(706), 3)
	require.NoError(t, err)
	item4, err := cards.NewCollectable(cards.NewID(10582), 1)
	require.NoError(t, err)
	item5, err := cards.NewCollectable(cards.NewID(582), 3)
	require.NoError(t, err)
	collected := map[string][]cards.Collectable{
		validClaim.ID: {item1, item2, item4, item5},
	}

	repo, err := memory.NewCardRepository(seed, collected)
	require.NoError(t, err)

	oCfg := auth.Config{}
	provider := auth.NewFakeProvider(auth.WithClaims(validClaim))
	authSvc := auth.New(oCfg, auth.NewProviders(provider))
	searchSvc := cards.NewCardService(repo)
	srv.RegisterRoutes(func(r fiber.Router) {
		cardsapi.SearchRoutes(r.Group("/"), web.NewAuthMiddleware(oCfg, authSvc), searchSvc)
	})

	return srv, provider
}
