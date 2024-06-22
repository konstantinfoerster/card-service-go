package detect

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/cards"
	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
	"github.com/konstantinfoerster/card-service-go/internal/common/web"
	"github.com/konstantinfoerster/card-service-go/internal/config"
)

func Routes(r fiber.Router, cfg config.Oidc, authSvc oidc.UserService, detectSvc Service) {
	relaxedAuthMiddleware := oidc.NewOauthMiddleware(authSvc, oidc.FromConfig(cfg), oidc.AllowUnauthorized())

	r.Post("/detect", relaxedAuthMiddleware, detect(detectSvc))
}

func detect(svc Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// when user is not set, the user specific collection data won't be loaded
		user, _ := auth.UserFromCtx(c)

		fHeader, err := c.FormFile("file")
		if err != nil {
			return aerrors.NewInvalidInputError(err, "invalid-file", "failed to read file from form")
		}

		file, err := fHeader.Open()
		if err != nil {
			return aerrors.NewInvalidInputError(err, "invalid-file", "failed to open file")
		}
		defer commonio.Close(file)

		result, err := svc.Detect(c.Context(), cards.AsCollector(user), file)
		if err != nil {
			return err
		}

		pagedResult := newMatchesResult(result)
		if web.AcceptsHTML(c) || web.IsHTMX(c) {
			data := fiber.Map{
				"Page": pagedResult,
			}

			return web.RenderPartial(c, "search", data)
		}

		return web.RenderJSON(c, pagedResult)
	}
}

func newMatchesResult(matches Matches) *web.PagedResponse[cards.CardDTO] {
	data := make([]cards.CardDTO, len(matches))
	for i, m := range matches {
		s := m.Score
		data[i] = cards.CardDTO{
			ItemDTO: cards.NewItemDTO(m.ID, m.Amount),
			Name:    m.Name,
			Image:   m.Image,
			Score:   &s,
		}
	}

	firstPage := 1
	nextPage := 2

	return &web.PagedResponse[cards.CardDTO]{
		Data:     data,
		HasMore:  false,
		Page:     firstPage,
		NextPage: nextPage,
	}
}
