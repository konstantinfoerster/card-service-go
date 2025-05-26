package cardsapi

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

const (
	detailsTmpl = "card_detail"
	printsTmpl  = "card_prints"
)

type CardService interface {
	Search(ctx context.Context, name string, collector cards.Collector, page cards.Page) (cards.Cards, error)
	Detail(ctx context.Context, id cards.ID, collector cards.Collector, page cards.Page) (cards.CardDetail, error)
}

func SearchRoutes(r fiber.Router, auth web.AuthMiddleware, searchSvc CardService) {
	log := slog.Default()
	r.Get("/cards", auth.Relaxed(), searchCards(searchSvc, log))
	r.Get("/cards/:id", auth.Relaxed(), details(searchSvc, detailsTmpl, log))
	r.Get("/cards/:id/prints", auth.Relaxed(), details(searchSvc, printsTmpl, log))
}

func searchCards(svc CardService, log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, _ := web.UserFromCtx(c)

		searchTerm := c.Query("name")
		page := newPage(c)
		log.Debug("search for card", slog.String("name", searchTerm), slog.Any("page", page))
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

		log.Debug("render json page",
			slog.Any("page", pagedResult.Page),
			slog.Bool("more", pagedResult.HasMore),
			slog.Int("size", len(pagedResult.Data)),
		)

		return web.RenderJSON(c, pagedResult)
	}
}

func details(svc CardService, tmplName string, log *slog.Logger) fiber.Handler {
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

		log.Debug("detail for card",
			slog.Group("card",
				slog.String("id", id.String()),
				slog.String("name", detail.Card.Name),
			),
			slog.Any("page", page),
		)
		data := fiber.Map{
			"Card":   newCard(detail.Card),
			"Prints": newPagedResponse(detail.Prints.PagedResult),
		}

		return web.RenderPartial(c, tmplName, data)
	}
}
