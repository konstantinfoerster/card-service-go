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
	r.Delete("/mycards", authMiddleware, remove(collectSvc))
}

func searchInPersonalCollection(svc application.CollectService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := auth.UserFromCtx(c)
		if err != nil {
			return err
		}

		result, err := svc.Search(c.Query("name"), newPage(c), collector(user))
		if err != nil {
			return err
		}

		return commonhttp.RenderJSON(c, newPagedResult(result))
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

		it, err := domain.NewItem(body.ID)
		if err != nil {
			return err
		}

		result, err := svc.Collect(it, collector(user))
		if err != nil {
			return err
		}

		return commonhttp.RenderJSON(c, newCollectableResult(result))
	}
}

func remove(svc application.CollectService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := auth.UserFromCtx(c)
		if err != nil {
			return common.NewAuthorizationError(err, "unauthorized")
		}

		var body Item
		if err = c.BodyParser(&body); err != nil {
			return common.NewInvalidInputMsg("invalid-body", "failed to parse body")
		}

		it, err := domain.NewItem(body.ID)
		if err != nil {
			return err
		}

		result, err := svc.Remove(it, collector(user))
		if err != nil {
			return err
		}

		return commonhttp.RenderJSON(c, newCollectableResult(result))
	}
}

type Item struct {
	ID int `json:"id"`
}

type CollectableResult struct {
	ID     int `json:"id"`
	Amount int `json:"amount"`
}

func newCollectableResult(r domain.CollectableResult) *CollectableResult {
	return &CollectableResult{
		ID:     r.ID,
		Amount: r.Amount,
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
