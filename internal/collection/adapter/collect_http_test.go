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
	collectedCards = append(collectedCards, &domain.Card{ID: 1, Name: "Dummy Card 1", Amount: 2})
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

func (s *fakeCollectService) Collect(item domain.Item, collector domain.Collector) (domain.CollectableResult, error) {
	if collector.ID == validUser {
		return domain.NewCollectableResult(item, 1), nil
	}

	return domain.CollectableResult{}, fmt.Errorf("invalid user")
}

func (s *fakeCollectService) Remove(item domain.Item, collector domain.Collector) (domain.CollectableResult, error) {
	if collector.ID == validUser {
		return domain.NewCollectableResult(item, 0), nil
	}

	return domain.CollectableResult{}, fmt.Errorf("invalid user")
}

func collectServer() *server.Server {
	srv := defaultServer()

	authSvc := NewFakeUserService()
	collectSvc := NewFakeCollectService()

	srv.RegisterAPIRoutes(func(r fiber.Router) {
		adapter.CollectRoutes(r.Group("/"), config.Oidc{}, authSvc, collectSvc)
	})

	return srv
}

func TestSearchCollected(t *testing.T) {
	srv := collectServer()
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/mycards?name=Dummy+Card"),
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
	assert.Equal(t, fiber.MIMEApplicationJSONCharsetUTF8, resp.Header.Get(fiber.HeaderContentType))
	assert.False(t, body.HasMore)
	assert.Equal(t, 1, body.Page)
	assert.Len(t, body.Data, 3)
	assert.Equal(t, &adapter.Card{ID: 1, Name: "Dummy Card 1", Amount: 2}, body.Data[0])
	assert.Equal(t, &adapter.Card{ID: 2, Name: "Dummy Card 2", Amount: 3}, body.Data[1])
	assert.Equal(t, &adapter.Card{ID: 3, Name: "Dummy Card 3", Amount: 5}, body.Data[2])
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
	body := commontest.FromJSON[adapter.CollectableResult](t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, fiber.MIMEApplicationJSONCharsetUTF8, resp.Header.Get(fiber.HeaderContentType))
	assert.Equal(t, &adapter.CollectableResult{ID: 1, Amount: 1}, body)
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

func TestRemoveItem(t *testing.T) {
	srv := collectServer()
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: validUser})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodDelete),
		commontest.WithURL("http://localhost/mycards"),
		commontest.WithEncryptedCookie(t, http.Cookie{
			Name:  "SESSION",
			Value: token,
		}),
		commontest.WithJSONBody(t, adapter.Item{ID: 1}),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)
	body := commontest.FromJSON[adapter.CollectableResult](t, resp)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, fiber.MIMEApplicationJSONCharsetUTF8, resp.Header.Get(fiber.HeaderContentType))
	assert.Equal(t, &adapter.CollectableResult{ID: 1, Amount: 0}, body)
}

func TestRemoveItemWithInvalidUser(t *testing.T) {
	srv := collectServer()
	token := commontest.Base64Encoded(t, &oidc.JSONWebToken{IDToken: invalidUser})
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodDelete),
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
