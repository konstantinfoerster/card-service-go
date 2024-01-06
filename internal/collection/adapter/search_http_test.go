package adapter_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/collection/adapter"
	"github.com/konstantinfoerster/card-service-go/internal/collection/application"
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
	"github.com/konstantinfoerster/card-service-go/internal/common/server"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSearchService struct {
	cards       []*domain.Card
	collections map[string][]*domain.Card
}

func NewFakeSearchService() application.SearchService {
	cards := make([]*domain.Card, 0)
	cards = append(cards, &domain.Card{ID: 1, Name: "Dummy Card 1"})
	cards = append(cards, &domain.Card{ID: 2, Name: "Dummy Card 2"})
	cards = append(cards, &domain.Card{ID: 3, Name: "Dummy Card 3"})

	collectedCards := make([]*domain.Card, 0)
	collectedCards = append(collectedCards, &domain.Card{ID: 1, Name: "Dummy Card 1", Amount: 2})
	collectedCards = append(collectedCards, &domain.Card{ID: 2, Name: "Dummy Card 2", Amount: 0})
	collectedCards = append(collectedCards, &domain.Card{ID: 3, Name: "Dummy Card 3", Amount: 5})

	collections := map[string][]*domain.Card{
		validUser: collectedCards,
	}

	return &fakeSearchService{
		cards:       cards,
		collections: collections,
	}
}

func (s *fakeSearchService) Search(name string, page domain.Page, collector domain.Collector) (domain.PagedResult, error) {
	result := make([]*domain.Card, 0)

	cards := s.cards
	if collector.ID != "" {
		cards = s.collections[collector.ID]
	}

	for _, c := range cards {
		if strings.Contains(c.Name, name) {
			result = append(result, c)
		}
	}

	return domain.NewPagedResult(result, page), nil
}

func TestSearch(t *testing.T) {
	srv := searchServer()
	cases := []struct {
		name                string
		header              func(req *http.Request)
		page                string
		expectedContentType string
		assertContent       func(t *testing.T, resp *http.Response)
	}{
		{
			name:                "json",
			header:              func(req *http.Request) {},
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.FromJSON[adapter.PagedResult](t, resp)
				assert.False(t, body.HasMore)
				assert.Equal(t, 1, body.Page)
				assert.Len(t, body.Data, 3)
				assert.Equal(t, &adapter.Card{Item: adapter.Item{ID: 1}, Name: "Dummy Card 1"}, body.Data[0])
				assert.Equal(t, &adapter.Card{Item: adapter.Item{ID: 2}, Name: "Dummy Card 2"}, body.Data[1])
				assert.Equal(t, &adapter.Card{Item: adapter.Item{ID: 3}, Name: "Dummy Card 3"}, body.Data[2])
			},
		},
		{
			name:                "json paged",
			header:              func(req *http.Request) {},
			page:                "size=1&page=2",
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.FromJSON[adapter.PagedResult](t, resp)
				assert.Equal(t, 2, body.Page)
				assert.True(t, body.HasMore)
			},
		},
		{
			name: "html",
			header: func(req *http.Request) {
				req.Header.Set(fiber.HeaderAccept, fiber.MIMETextHTMLCharsetUTF8)
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsFullHTML(t, body)
				commontest.AssertContainsLogin(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name: "htmx",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name: "htmx paged",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			page:                "size=1&page=2",
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsPartialHTML(t, body)
				assert.NotContains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
			},
		},
		{
			name: "htmx with lazy loading marker",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			page:                "size=1",
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				assert.Contains(t, body, "class=\"last-img\"")
			},
		},
		{
			name: "htmx without lazy loading marker",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				assert.NotContains(t, body, "class=\"last-img\"")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := commontest.NewRequest(
				commontest.WithMethod(http.MethodGet),
				commontest.WithURL("http://localhost/cards?name=Dummy+Card&"+tc.page),
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

func TestSearchWithUser(t *testing.T) {
	srv := searchServer()
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser})
	cases := []struct {
		name                string
		header              func(req *http.Request)
		expectedContentType string
		assertContent       func(t *testing.T, resp *http.Response)
	}{
		{
			name:                "json",
			header:              func(req *http.Request) {},
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.FromJSON[adapter.PagedResult](t, resp)
				assert.False(t, body.HasMore)
				assert.Equal(t, 1, body.Page)
				assert.Len(t, body.Data, 3)
				assert.Equal(t, &adapter.Card{Item: adapter.Item{ID: 1, Amount: 2}, Name: "Dummy Card 1"}, body.Data[0])
				assert.Equal(t, &adapter.Card{Item: adapter.Item{ID: 2, Amount: 0}, Name: "Dummy Card 2"}, body.Data[1])
				assert.Equal(t, &adapter.Card{Item: adapter.Item{ID: 3, Amount: 5}, Name: "Dummy Card 3"}, body.Data[2])
			},
		},
		{
			name: "html",
			header: func(req *http.Request) {
				req.Header.Set(fiber.HeaderAccept, fiber.MIMETextHTMLCharsetUTF8)
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsFullHTML(t, body)
				commontest.AssertContainsProfile(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
				assert.Equal(t, 3, strings.Count(body, "data-testid=\"add-card-btn\""))
				assert.Equal(t, 2, strings.Count(body, "data-testid=\"remove-card-btn\""))
			},
		},
		{
			name: "htmx",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsPartialHTML(t, body)
				assert.Contains(t, body, "data-testid=\"search-result-txt\"")
				assert.Equalf(t, 3, strings.Count(body, "data-testid=\"card-"), "expected 3 cards in %s", body)
				assert.Equal(t, 3, strings.Count(body, "data-testid=\"add-card-btn\""))
				assert.Equal(t, 2, strings.Count(body, "data-testid=\"remove-card-btn\""))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := commontest.NewRequest(
				commontest.WithMethod(http.MethodGet),
				commontest.WithURL("http://localhost/cards?name=Dummy+Card"),
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

func TestSearchWithInvalidUser(t *testing.T) {
	srv := searchServer()
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: invalidUser})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/cards?name=Dummy+Card&size=5&page=1"),
		commontest.WithEncryptedCookie(t, http.Cookie{
			Name:  "SESSION",
			Value: token,
		}),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func searchServer() *server.Server {
	srv := defaultServer()

	authSvc := NewFakeUserService()
	searchSvc := NewFakeSearchService()
	srv.RegisterRoutes(func(r fiber.Router) {
		adapter.SearchRoutes(r.Group("/"), config.Oidc{}, authSvc, searchSvc)
	})

	return srv
}
