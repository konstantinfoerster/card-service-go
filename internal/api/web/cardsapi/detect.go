package cardsapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/aio"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

func DetectRoutes(r fiber.Router, auth web.AuthMiddleware, detectSvc cards.DetectService) {
	r.Post("/detect", auth.Relaxed(), Detect(detectSvc))
}

func Detect(svc cards.DetectService) fiber.Handler {
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

		pagedResult := newResponse(result.PagedResult)
		if web.AcceptsHTML(c) || web.IsHTMX(c) {
			data := fiber.Map{
				"Page": pagedResult,
			}

			return web.RenderPartial(c, "search", data)
		}

		return web.RenderJSON(c, pagedResult)
	}
}
