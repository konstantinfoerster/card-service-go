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
		return c.JSON(&fiber.Map{
			"has_more": result.HasMore,
			"total":    result.Total,
			"data":     result.Result,
		})
	}
}
