package collection

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/web"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

func Routes(r fiber.Router, cfg config.Oidc, oSvc oidc.UserService, cSvc Service) {
	authMiddleware := oidc.NewOauthMiddleware(oSvc, oidc.FromConfig(cfg))

	r.Get("/mycards", authMiddleware, searchInPersonalCollection(cSvc))
	r.Post("/mycards", authMiddleware, collect(cSvc))
}

func searchInPersonalCollection(svc Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := auth.UserFromCtx(c)
		if err != nil {
			return aerrors.NewAuthorizationError(err, "unauthorized")
		}

		searchTerm := c.Query("name")

		result, err := svc.Search(c.Context(), searchTerm, cards.AsCollector(user), web.NewPage(c))
		if err != nil {
			return err
		}

		pagedResult := cards.NewPagedResponse(result)

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

func collect(svc Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := auth.UserFromCtx(c)
		if err != nil {
			return aerrors.NewAuthorizationError(err, "unauthorized")
		}

		var body cards.ItemDTO
		if err = c.BodyParser(&body); err != nil {
			return aerrors.NewInvalidInputMsg("invalid-body", "failed to parse body")
		}

		it, err := NewItem(body.ID, body.Amount)
		if err != nil {
			return err
		}

        item, err := svc.Collect(c.Context(), it, cards.AsCollector(user))
		if err != nil {
			return err
		}

		if web.IsHTMX(c) {
			return web.RenderPartial(c, "collect_action", cards.NewItemDTO(item.ID, item.Amount))
		}

		return web.RenderJSON(c, item)
	}
}
