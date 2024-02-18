package adapter

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/collection/application"
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
)

func CollectRoutes(r fiber.Router, cfg config.Oidc, oidcSvc oidc.UserService, collectSvc application.CollectService) {
	authMiddleware := oidc.NewOauthMiddleware(oidcSvc, oidc.FromConfig(cfg))

	r.Get("/mycards", authMiddleware, searchInPersonalCollection(collectSvc))
	r.Post("/mycards", authMiddleware, collect(collectSvc))
}

func searchInPersonalCollection(svc application.CollectService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := auth.UserFromCtx(c)
		if err != nil {
			return err
		}

		searchTerm := c.Query("name")

		result, err := svc.Search(searchTerm, newPage(c), collector(user))
		if err != nil {
			return err
		}

		pagedResult := newPagedResult(result)

		if commonhttp.AcceptsHTML(c) || commonhttp.IsHTMX(c) {
			data := fiber.Map{
				"Query": Query{
					Name: searchTerm,
				},
				"Page": pagedResult,
			}

			if commonhttp.IsHTMX(c) {
				if c.Query("page") == "" {
					return commonhttp.RenderPartial(c, "mycards", data)
				}

				return commonhttp.RenderPartial(c, "card_list", data)
			}

			return commonhttp.RenderPage(c, "mycards", data)
		}

		return commonhttp.RenderJSON(c, pagedResult)
	}
}

func collect(svc application.CollectService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := auth.UserFromCtx(c)
		if err != nil {
			return common.NewAuthorizationError(err, "unauthorized")
		}

		var body Item
		if err = c.BodyParser(&body); err != nil {
			return common.NewInvalidInputMsg("invalid-body", "failed to parse body")
		}

		it, err := domain.NewItem(body.ID, body.Amount)
		if err != nil {
			return err
		}

		item, err := svc.Collect(it, collector(user))
		if err != nil {
			return err
		}

		if commonhttp.IsHTMX(c) {
			return commonhttp.RenderPartial(c, "collect_action", newItem(item.ID, item.Amount))
		}

		return commonhttp.RenderJSON(c, item)
	}
}

func collector(user *auth.User) domain.Collector {
	if user == nil {
		return domain.Collector{}
	}

	return domain.Collector{
		ID: user.ID,
	}
}
