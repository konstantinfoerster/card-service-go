package cardsapi

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

func CollectionRoutes(r fiber.Router, auth web.AuthMiddleware, cSvc cards.CollectionService) {
	r.Get("/mycards", auth.Required(), searchInPersonalCollection(cSvc))
	r.Post("/mycards", auth.Required(), collect(cSvc))
}

func searchInPersonalCollection(svc cards.CollectionService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := web.UserFromCtx(c)
		if err != nil {
			return aerrors.NewAuthorizationError(err, "unauthorized")
		}

		searchTerm := c.Query("name")
		page := newPage(c)
		result, err := svc.Search(c.Context(), searchTerm, cards.NewCollector(user.ID), page)
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
					return web.RenderPartial(c, "mycards", data)
				}

				return web.RenderPartial(c, "card_list", data)
			}

			return web.RenderPage(c, "mycards", data)
		}

		return web.RenderJSON(c, pagedResult)
	}
}

func collect(svc cards.CollectionService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := web.UserFromCtx(c)
		if err != nil {
			return aerrors.NewAuthorizationError(err, "unauthorized")
		}

		var body Item
		if err = c.BodyParser(&body); err != nil {
			return aerrors.NewInvalidInputError(err, "invalid-body", "invalid body format")
		}
		id, err := toID(body.ID)
		if err != nil {
			return aerrors.NewInvalidInputMsg("invalid-id", "invalid id format")
		}
		it, err := cards.NewCollectable(id, body.Amount.Value())
		if err != nil {
			return err
		}

		item, err := svc.Collect(c.Context(), it, cards.NewCollector(user.ID))
		if err != nil {
			return err
		}

		result := NewItem(item.ID, item.Amount)
		if web.IsHTMX(c) {
			return web.RenderPartial(c, "collect_action", result)
		}

		return web.RenderJSON(c, result)
	}
}
