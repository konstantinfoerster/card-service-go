package cardsapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type DetectService interface {
	Detect(ctx context.Context, collector cards.Collector, in io.Reader) (cards.Matches, error)
	DetectByHash(ctx context.Context, collector cards.Collector, hashes ...cards.Hash) (cards.Matches, error)
}

func DetectRoutes(r fiber.Router, cfg web.Auth, detectSvc DetectService) {
	authHandler := web.NewMiddleware(cfg, web.AllowUnauthorized())

	r.Post("/detect", authHandler, detect(detectSvc))
	r.Get("/detect/live", authHandler, detectLive(cfg))
	// r.Get("/detect/:hash", authHandler, detectByHash(detectSvc))
}

func detect(svc DetectService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, _ := web.UserFromCtx(c)

		var req struct {
			Image string `json:"image"`
		}
		if err := c.BodyParser(&req); err != nil {
			return err
		}

		imgBytes, err := base64.StdEncoding.DecodeString(req.Image)
		if err != nil {
			return err
		}
		result, err := svc.Detect(c.Context(), asCollector(user.ID), bytes.NewReader(imgBytes))
		if err != nil {
			return err
		}

		slog.Info("detected", slog.Int("size", result.Size))

		pagedResult := newPagedResponse(result.PagedResult)
		return web.RenderJSON(c, pagedResult)
	}
}

func detectByHash(svc DetectService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, _ := web.UserFromCtx(c)

		sHash := c.Params("hash")
		if sHash == "" {
			emptyMatches := cards.EmptyMatches(cards.DefaultPage())
			return web.RenderJSON(c, newPagedResponse(emptyMatches.PagedResult))
		}

		hash, err := cards.FromHexString(sHash)
		if err != nil {
			return err
		}

		result, err := svc.DetectByHash(c.Context(), asCollector(user.ID), hash)
		if err != nil {
			return err
		}

		pagedResult := newPagedResponse(result.PagedResult)

		return web.RenderJSON(c, pagedResult)
	}
}

func detectLive(cfg web.Auth) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if web.IsHTMX(c) {
			return web.RenderPartial(c, "detect", nil)
		}

		return web.RenderPage(c, cfg, "detect", nil)
	}
}
