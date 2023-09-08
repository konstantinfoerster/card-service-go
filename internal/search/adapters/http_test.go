package adapters_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common/server"
	commontest "github.com/konstantinfoerster/card-service-go/internal/common/test"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/konstantinfoerster/card-service-go/internal/search/adapters"
	"github.com/konstantinfoerster/card-service-go/internal/search/application"
	"github.com/konstantinfoerster/card-service-go/internal/search/domain"
	"github.com/stretchr/testify/assert"
)

func NewFakeApplication() application.Service {
	cards := make([]*domain.Card, 0)
	cards = append(cards, &domain.Card{Name: "Dummy Card 1", Image: "http://localhost/dummyCard1.jpeg"})
	cards = append(cards, &domain.Card{Name: "Dummy Card 2", Image: "http://localhost/dummyCard2.jpeg"})
	cards = append(cards, &domain.Card{Name: "Dummy Card 3", Image: "http://localhost/dummyCard3.jpeg"})

	return &FakeApplicationService{cards: cards}
}

type FakeApplicationService struct {
	cards []*domain.Card
}

func (s *FakeApplicationService) SimpleSearch(name string, page domain.Page) (domain.PagedResult, error) {
	result := make([]*domain.Card, 0)
	for _, c := range s.cards {
		if strings.Contains(c.Name, name) {
			result = append(result, c)
		}
	}

	return domain.NewPagedResult(result, page), nil
}

func TestSearchDefaultPage(t *testing.T) {
	srv := defaultServer()
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/search?name=Dummy+Card"),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)
	body := commontest.FromJSON[adapters.PagedResult](t, resp)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, fiber.MIMEApplicationJSONCharsetUTF8, resp.Header.Get(fiber.HeaderContentType))
	assert.Equal(t, 1, body.Page)
	assert.Equal(t, false, body.HasMore)
	assert.Len(t, body.Data, 3)
	assert.Equal(t, "Dummy Card 1", body.Data[0].Name)
	assert.Equal(t, "Dummy Card 2", body.Data[1].Name)
	assert.Equal(t, "Dummy Card 3", body.Data[2].Name)
}

func TestSearchPaged(t *testing.T) {
	srv := defaultServer()
	req := commontest.NewRequest(
		commontest.WithMethod(http.MethodGet),
		commontest.WithURL("http://localhost/search?name=Dummy+Card&size=1&page=2"),
	)

	resp, err := srv.Test(req)
	defer commontest.Close(t, resp)
	body := commontest.FromJSON[adapters.PagedResult](t, resp)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, body.Page)
	assert.Equal(t, true, body.HasMore)
}

func defaultServer() *server.Server {
	cfg := &config.Config{
		Server: config.Server{
			Cookie: config.Cookie{
				EncryptionKey: "01234567890123456789012345678901",
			},
		},
	}
	svc := NewFakeApplication()
	srv := server.NewHTTPServer(&cfg.Server)

	srv.RegisterAPIRoutes(func(r fiber.Router) {
		adapters.Routes(r.Group("/"), svc)
	})

	return srv
}
