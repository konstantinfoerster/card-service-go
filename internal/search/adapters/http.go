package adapters

import (
	"github.com/gofiber/fiber/v2"
	httperr "github.com/konstantinfoerster/card-service/internal/common/http"
	"github.com/konstantinfoerster/card-service/internal/search/domain"
	"github.com/konstantinfoerster/card-service/internal/search/service"
	"strconv"
)

func NewPage(c *fiber.Ctx) domain.Page {
	size, _ := strconv.Atoi(c.Query("size", "0"))
	page, _ := strconv.Atoi(c.Query("page", "0"))
	return domain.NewPage(page, size)
}

func SimpleSearch(service service.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := service.SimpleSearch(c.Query("name"), NewPage(c))
		if err != nil {
			return httperr.RespondWithProblemJson(err, c)
		}
		return c.JSON(NewPagedResult(result))
	}
}
func NewPagedResult(pr domain.PagedResult) *PagedResult {
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
		Total:   pr.Total,
		Page:    pr.Page,
	}
}

type PagedResult struct {
	Data    []*Card `json:"data"`
	HasMore bool    `json:"has_more"`
	Total   int     `json:"total"`
	Page    int     `json:"page"`
}

type Card struct {
	Name  string `json:"name"`
	Image string `json:"image,omitempty"`
}
