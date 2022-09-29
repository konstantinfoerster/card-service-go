package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service/internal/common/errors"
	"github.com/konstantinfoerster/card-service/internal/search/domain"
	"github.com/konstantinfoerster/card-service/internal/search/service"
	"github.com/rs/zerolog/log"
	"net/http"
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
			return respondWithProblemJson(err, c)
		}
		return c.JSON(&fiber.Map{
			"has_more": result.HasMore,
			"total":    result.Total,
			"data":     result.Result,
		})
	}
}

func respondWithProblemJson(err error, c *fiber.Ctx) error {
	appError, ok := err.(errors.Error)
	if !ok {
		return InternalError(http.StatusText(http.StatusInternalServerError), appError, c)
	}
	log.Error().Err(appError).Str("key", appError.Key()).Send()
	switch appError.Type() {
	default:
		return InternalError(appError.Key(), appError, c)
	}
}

func InternalError(title string, err error, c *fiber.Ctx) error {
	log.Error().Err(err).Send()
	return c.Status(http.StatusInternalServerError).
		Type("application/problem+json").
		JSON(&fiber.Map{
			"title":  title,
			"status": http.StatusInternalServerError,
		})
}
