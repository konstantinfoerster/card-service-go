package cardsapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
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
		defer commonio.Close(file)

		result, err := svc.Detect(c.Context(), asCollector(user), file)
		if err != nil {
			return err
		}

		pagedResult := newMatchesResponse(result)
		if web.AcceptsHTML(c) || web.IsHTMX(c) {
			data := fiber.Map{
				"Page": pagedResult,
			}

			return web.RenderPartial(c, "search", data)
		}

		return web.RenderJSON(c, pagedResult)
	}
}

func newMatchesResponse(matches cards.Matches) *PagedResponse[Card] {
	data := make([]Card, len(matches))
	for i, m := range matches {
		s := m.Score
		data[i] = Card{
			Item:  newItem(m.ID, m.Amount),
			Name:  m.Name,
			Image: m.Image,
			Score: &s,
		}
	}

	firstPage := 1
	nextPage := 2

	return &PagedResponse[Card]{
		Data:     data,
		HasMore:  false,
		Page:     firstPage,
		NextPage: nextPage,
	}
}
