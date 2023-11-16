package adapter

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/collection/application"
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
)

func SearchRoutes(r fiber.Router, cfg config.Oidc, authSvc oidc.UserService, searchSvc application.SearchService) {
	relaxedAuthMiddleware := oidc.NewOauthMiddleware(authSvc, oidc.FromConfig(cfg), oidc.AllowEmptyCookie())

	r.Get("/cards", relaxedAuthMiddleware, search(searchSvc))
}

func search(svc application.SearchService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// when user is not set, the user specific collection data won't be loaded
		user, _ := auth.UserFromCtx(c)

		result, err := svc.Search(c.Query("name"), newPage(c), collector(user))
		if err != nil {
			return err
		}

		return commonhttp.RenderJSON(c, newPagedResult(result))
	}
}

func newPage(c *fiber.Ctx) domain.Page {
	size, _ := strconv.Atoi(c.Query("size", ""))
	page, _ := strconv.Atoi(c.Query("page", "0"))

	return domain.NewPage(page, size)
}

func newPagedResult(pr domain.PagedResult) *PagedResult {
	data := make([]*Card, len(pr.Result))
	for i, c := range pr.Result {
		data[i] = &Card{
			ID:     c.ID,
			Name:   c.Name,
			Image:  c.Image,
			Amount: c.Amount,
		}
	}

	return &PagedResult{
		Data:    data,
		HasMore: pr.HasMore,
		Page:    pr.Page,
	}
}

type PagedResult struct {
	Data    []*Card `json:"data"`
	HasMore bool    `json:"hasMore"`
	Page    int     `json:"page"`
}

type Card struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image,omitempty"`
	Amount int    `json:"amount,omitempty"`
}
