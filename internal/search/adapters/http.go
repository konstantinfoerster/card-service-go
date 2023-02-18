package adapters

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common/problemjson"
	"github.com/konstantinfoerster/card-service-go/internal/search/application"
	"github.com/konstantinfoerster/card-service-go/internal/search/domain"
)

func NewPage(c *fiber.Ctx) domain.Page {
	size, _ := strconv.Atoi(c.Query("size", "0"))
	page, _ := strconv.Atoi(c.Query("page", "0"))

	return domain.NewPage(page, size)
}

func SimpleSearch(service application.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := service.SimpleSearch(c.Query("name"), NewPage(c))
		if err != nil {
			return problemjson.RespondWithProblemJSON(err, c)
		}

		if err = c.JSON(NewPagedResult(result)); err != nil {
			return problemjson.RespondWithProblemJSON(fmt.Errorf("failed to encode search result, %w", err), c)
		}

		return nil
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
	HasMore bool    `json:"hasMore"`
	Total   int     `json:"total"`
	Page    int     `json:"page"`
}

type Card struct {
	Name  string `json:"name"`
	Image string `json:"image,omitempty"`
}
