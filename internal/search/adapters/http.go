package adapters

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
	"github.com/konstantinfoerster/card-service-go/internal/search/application"
	"github.com/konstantinfoerster/card-service-go/internal/search/domain"
)

func Routes(r fiber.Router, appSvc application.Service) {
	r.Get("/search", SimpleSearch(appSvc))
}

func SimpleSearch(service application.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := service.SimpleSearch(c.Query("name"), newPage(c))
		if err != nil {
			return err
		}

		return commonhttp.RenderJSON(c, newPagedResult(result))
	}
}

func newPage(c *fiber.Ctx) domain.Page {
	size, _ := strconv.Atoi(c.Query("size", ""))
	page, _ := strconv.Atoi(c.Query("page", "0"))

	return domain.NewPage(page, size)
}

func newPagedResult(pr domain.PagedResult) *PagedResult {
	data := make([]*Card, len(pr.Result))
	for i, c := range pr.Result {
		data[i] = &Card{
			Image: c.Image,
			Name:  c.Name,
		}
	}

	return &PagedResult{
		Data:    data,
		HasMore: pr.HasMore,
		Page:    pr.Page,
	}
}

type PagedResult struct {
	Data    []*Card `json:"data"`
	HasMore bool    `json:"hasMore"`
	Page    int     `json:"page"`
}

type Card struct {
	Name  string `json:"name"`
	Image string `json:"image,omitempty"`
}
