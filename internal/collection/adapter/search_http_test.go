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
	collectedCards = append(collectedCards, &domain.Card{ID: 2, Name: "Dummy Card 2", Amount: 3})
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

func TestSearchDefaultPage(t *testing.T) {
	srv := searchServer()
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/cards?name=Dummy+Card"),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)
	body := commontest.FromJSON[adapter.PagedResult](t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, fiber.MIMEApplicationJSONCharsetUTF8, resp.Header.Get(fiber.HeaderContentType))
	assert.False(t, body.HasMore)
	assert.Equal(t, 1, body.Page)
	assert.Len(t, body.Data, 3)
	assert.Equal(t, &adapter.Card{ID: 1, Name: "Dummy Card 1"}, body.Data[0])
	assert.Equal(t, &adapter.Card{ID: 2, Name: "Dummy Card 2"}, body.Data[1])
	assert.Equal(t, &adapter.Card{ID: 3, Name: "Dummy Card 3"}, body.Data[2])
}

func TestSearchPaged(t *testing.T) {
	srv := searchServer()
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/cards?name=Dummy+Card&size=1&page=2"),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)
	body := commontest.FromJSON[adapter.PagedResult](t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, body.Page)
	assert.True(t, body.HasMore)
}

func TestSearchWithUser(t *testing.T) {
	srv := searchServer()
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser})
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
	body := commontest.FromJSON[adapter.PagedResult](t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, &adapter.Card{ID: 1, Name: "Dummy Card 1", Amount: 2}, body.Data[0])
	assert.Equal(t, &adapter.Card{ID: 2, Name: "Dummy Card 2", Amount: 3}, body.Data[1])
	assert.Equal(t, &adapter.Card{ID: 3, Name: "Dummy Card 3", Amount: 5}, body.Data[2])
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
	srv.RegisterAPIRoutes(func(r fiber.Router) {
		adapter.SearchRoutes(r.Group("/"), config.Oidc{}, authSvc, searchSvc)
	})

	return srv
}
