package cardsapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
)

type DetectService interface {
	Detect(ctx context.Context, collector cards.Collector, in io.Reader) (cards.Matches, error)
}

func DetectRoutes(r fiber.Router, cfg web.Auth, srvCfg web.Config, detectSvc DetectService) {
	authHandler := web.NewMiddleware(cfg, web.AllowUnauthorized())

	r.Post("/detect", authHandler, detect(detectSvc, srvCfg))
	r.Get("/detect/live", authHandler, detectLive(cfg))
}

func detect(svc DetectService, cfg web.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, _ := web.UserFromCtx(c)

		if !c.Is("jpeg") && !c.Is("png") {
			return aerrors.NewInvalidInputMsg("invalid-content-type", "expected an image/jpeg or image/png body")
		}

		imgBytes := c.Body()
		if len(imgBytes) == 0 {
			return aerrors.NewInvalidInputMsg("invalid-body", "no image provided")
		}

		if cfg.Debug {
			saveDebugImage(imgBytes, cfg.DebugDir)
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

// saveDebugImage persists the given file as jpg in given dir.
func saveDebugImage(imgBytes []byte, dir string) {
	filePath := filepath.Join(dir, fmt.Sprintf("detect-%d.jpg", time.Now().UnixNano()))
	if err := os.WriteFile(filePath, imgBytes, 0600); err != nil {
		slog.Warn("failed to save debug image", slog.Any("error", err))

		return
	}

	slog.Info("saved debug image", slog.String("path", filePath))
}

func detectLive(cfg web.Auth) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// always make a full page load
		// otherwise the js files are re-executed very time and
		// that needs to be handled
		return web.RenderPage(c, cfg, "detect", nil)
	}
}
