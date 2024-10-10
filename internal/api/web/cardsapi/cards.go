package cardsapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/rs/zerolog/log"
)

func SearchRoutes(r fiber.Router, auth web.AuthMiddleware, searchSvc cards.CardService) {
	r.Get("/cards", auth.Relaxed(), searchCards(searchSvc))
}

func searchCards(svc cards.CardService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// when user is not set, the user specific collection data won't be loaded
		user, _ := web.UserFromCtx(c)

		searchTerm := c.Query("name")
		page := newPage(c)
		log.Debug().Msgf("search for card with name %s on page %#v", searchTerm, page)
		result, err := svc.Search(c.Context(), searchTerm, asCollector(user), page)
		if err != nil {
			return err
		}

		pagedResult := newResponse(result.PagedResult)

		if web.AcceptsHTML(c) || web.IsHTMX(c) {
			data := fiber.Map{
				"SearchTerm": searchTerm,
				"Page":       pagedResult,
			}

			if web.IsHTMX(c) {
				if c.Query("page") == "" {
					return web.RenderPartial(c, "search", data)
				}

				return web.RenderPartial(c, "card_list", data)
			}

			return web.RenderPage(c, "search", data)
		}

		log.Debug().Msgf("render json page %v, more = %v, size = %d",
			pagedResult.Page, pagedResult.HasMore, len(pagedResult.Data))

		return web.RenderJSON(c, pagedResult)
	}
}
