package adapter_test

import (
	"fmt"
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

type fakeCollectService struct {
	collections map[string][]*domain.Card
}

func NewFakeCollectService() application.CollectService {
	collectedCards := make([]*domain.Card, 0)
	collectedCards = append(collectedCards, &domain.Card{ID: 1, Name: "Dummy Card 1", Amount: 0})
	collectedCards = append(collectedCards, &domain.Card{ID: 2, Name: "Dummy Card 2", Amount: 3})
	collectedCards = append(collectedCards, &domain.Card{ID: 3, Name: "Dummy Card 3", Amount: 5})

	collections := map[string][]*domain.Card{
		validUser: collectedCards,
	}

	return &fakeCollectService{
		collections: collections,
	}
}

func (s *fakeCollectService) Search(name string, page domain.Page, collector domain.Collector) (domain.PagedResult, error) {
	result := make([]*domain.Card, 0)

	collected := s.collections[collector.ID]
	for _, c := range collected {
		if strings.Contains(c.Name, name) {
			result = append(result, c)
		}
	}

	return domain.NewPagedResult(result, page), nil
}

func (s *fakeCollectService) Collect(item domain.Item, collector domain.Collector) (domain.Item, error) {
	if collector.ID == validUser {
		return item, nil
	}

	return domain.Item{}, fmt.Errorf("invalid user")
}

func collectServer() *server.Server {
	srv := defaultServer()

	authSvc := NewFakeUserService()
	collectSvc := NewFakeCollectService()

	srv.RegisterRoutes(func(r fiber.Router) {
		adapter.CollectRoutes(r.Group("/"), config.Oidc{}, authSvc, collectSvc)
	})

	return srv
}

func TestSearchCollected(t *testing.T) {
	srv := collectServer()
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser})
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
				assert.Equal(t, &adapter.Card{Item: adapter.Item{ID: 1, Amount: 0}, Name: "Dummy Card 1"}, body.Data[0])
				assert.Equal(t, &adapter.Card{Item: adapter.Item{ID: 2, Amount: 3}, Name: "Dummy Card 2"}, body.Data[1])
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
				commontest.WithURL("http://localhost/mycards?name=Dummy+Card&"+tc.page),
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
	srv := collectServer()
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/mycards?name=Dummy+Card"),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestCollectItem(t *testing.T) {
	srv := collectServer()
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser})
	cases := []struct {
		name                string
		header              func(req *http.Request)
		amount              int
		expectedContentType string
		assertContent       func(t *testing.T, resp *http.Response)
	}{
		{
			name:                "json add",
			header:              func(req *http.Request) {},
			amount:              1,
			expectedContentType: fiber.MIMEApplicationJSONCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.FromJSON[adapter.Item](t, resp)
				assert.Equal(t, &adapter.Item{ID: 1, Amount: 1}, body)
			},
		},
		{
			name: "htmx add",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			amount:              1,
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsPartialHTML(t, body)

				assert.Contains(t, body, "data-testid=\"add-card-btn\"", "expect to have add button")
				assert.Contains(t, body, "data-testid=\"remove-card-btn\"", "expect to have remove button")
				assert.Contains(t, body, "hx-vals='{ \"id\": 1,\"amount\": 2 }'", "expect to find add values")
				assert.Contains(t, body, "hx-vals='{ \"id\": 1,\"amount\": 0 }'", "expect to find remove values")
			},
		},
		{
			name: "htmx remove",
			header: func(req *http.Request) {
				req.Header.Set(commonhttp.HeaderHTMXRequest, "true")
			},
			amount:              0,
			expectedContentType: fiber.MIMETextHTMLCharsetUTF8,
			assertContent: func(t *testing.T, resp *http.Response) {
				body := commontest.ToString(t, resp)
				commontest.AssertContainsPartialHTML(t, body)

				assert.Contains(t, body, "data-testid=\"add-card-btn\"", "expect to have add button")
				assert.NotContains(t, body, "data-testid=\"remove-card-btn\"", "expect to have disabled remove button")
				assert.Contains(t, body, "hx-vals='{ \"id\": 1,\"amount\": 1 }'", "expect to find add values")
				assert.Contains(t, body, "hx-vals='{ \"id\": 1,\"amount\": 0 }'", "expect to find remove values")
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

func TestCollectItemWithInvalidUser(t *testing.T) {
	srv := collectServer()
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
