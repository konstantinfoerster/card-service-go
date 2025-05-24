package cardsapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/rs/zerolog/log"
)

const (
	detailsTmpl = "card_detail"
	printsTmpl  = "card_prints"
)

func SearchRoutes(r fiber.Router, auth web.AuthMiddleware, searchSvc cards.CardService) {
	r.Get("/cards", auth.Relaxed(), searchCards(searchSvc))
	r.Get("/cards/:id", auth.Relaxed(), details(searchSvc, detailsTmpl))
	r.Get("/cards/:id/prints", auth.Relaxed(), details(searchSvc, printsTmpl))
}

func searchCards(svc cards.CardService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, _ := web.UserFromCtx(c)

		searchTerm := c.Query("name")
		page := newPage(c)
		log.Debug().Any("page", page).Msgf("search for card with name %s", searchTerm)
		result, err := svc.Search(c.Context(), searchTerm, asCollector(user), page)
		if err != nil {
			return err
		}

		pagedResult := newPagedResponse(result.PagedResult)

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

func details(svc cards.CardService, tmplName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !web.IsHTMX(c) {
			return aerrors.NewInvalidInputMsg("invalid-accept-header", "only htmlx supported")
		}

		user, _ := web.UserFromCtx(c)
		id, err := toID(c.Params("id"))
		if err != nil {
			return aerrors.NewInvalidInputError(err, "invalid-id", "invalid id")
		}

		page := newPage(c)
		detail, err := svc.Detail(c.Context(), id, asCollector(user), page)
		if err != nil {
			return err
		}

		log.Debug().Any("page", page).
			Msgf("detail for card %v name %s with %d prints", id, detail.Card.Name, detail.Prints.Size)
		data := fiber.Map{
			"Card":   newCard(detail.Card),
			"Prints": newPagedResponse(detail.Prints.PagedResult),
		}

		return web.RenderPartial(c, tmplName, data)
	}
}
