package cardsapi

import (
	"context"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/aio"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type DetectService interface {
	Detect(ctx context.Context, collector cards.Collector, in io.Reader) (cards.Matches, error)
}

func DetectRoutes(r fiber.Router, auth web.AuthMiddleware, detectSvc DetectService) {
	r.Post("/detect", auth.Relaxed(), Detect(detectSvc))
}

func Detect(svc DetectService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// when user is not set, the user specific collection data won't be loaded
		user, _ := web.UserFromCtx(c)

		fHeader, err := c.FormFile("file")
		if err != nil {
			return aerrors.NewInvalidInputError(err, "invalid-file", "failed to read file from form")
		}

		file, err := fHeader.Open()
		if err != nil {
			return aerrors.NewInvalidInputError(err, "invalid-file", "failed to open file")
		}
		defer aio.Close(file)

		result, err := svc.Detect(c.Context(), asCollector(user), file)
		if err != nil {
			return err
		}

		pagedResult := newPagedResponse(result.PagedResult)
		if web.AcceptsHTML(c) || web.IsHTMX(c) {
			data := fiber.Map{
				"Page": pagedResult,
			}

			return web.RenderPartial(c, "search", data)
		}

		return web.RenderJSON(c, pagedResult)
	}
}
