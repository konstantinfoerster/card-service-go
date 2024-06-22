package cards

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/web"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/rs/zerolog/log"
)

func Routes(r fiber.Router, cfg config.Oidc, authSvc oidc.UserService, searchSvc Searcher) {
	relaxedAuthMiddleware := oidc.NewOauthMiddleware(authSvc, oidc.FromConfig(cfg), oidc.AllowUnauthorized())

	r.Get("/cards", relaxedAuthMiddleware, search(searchSvc))


    // /cards
    // /mycards
    // /detect
}

func search(svc Searcher) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// when user is not set, the user specific collection data won't be loaded
		user, _ := auth.UserFromCtx(c)

		searchTerm := c.Query("name")
		page := web.NewPage(c)
		log.Debug().Msgf("search for card with name %s on page %#v", searchTerm, page)
		result, err := svc.Search(c.Context(), searchTerm, AsCollector(user), page)
		if err != nil {
			return err
		}

		pagedResult := NewPagedResponse(result)
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

func NewPagedResponse(pr Cards) *web.PagedResponse[CardDTO] {
	data := make([]CardDTO, len(pr.Result))
	for i, c := range pr.Result {
		data[i] = CardDTO{
			ItemDTO: NewItemDTO(c.ID, c.Amount),
			Name:    c.Name,
			Image:   c.Image,
		}
	}

	return &web.PagedResponse[CardDTO]{
		Data:     data,
		HasMore:  pr.HasMore,
		Page:     pr.Page,
		NextPage: pr.Page + 1,
	}
}

type CardDTO struct {
	Score *int   `json:"score,omitempty"`
	Name  string `json:"name"`
	Image string `json:"image,omitempty"`
	ItemDTO
}

func (c CardDTO) WithScore(v int) CardDTO {
	c.Score = &v

	return c
}

func NewItemDTO(id int, amount int) ItemDTO {
	return ItemDTO{
		ID:     id,
		Amount: amount,
	}
}

type ItemDTO struct {
	ID     int `json:"id"`
	Amount int `json:"amount,omitempty"`
}

func (i ItemDTO) NextAmount() int {
	return i.Amount + 1
}

func (i ItemDTO) PreviousAmount() int {
	prev := i.Amount - 1
	if prev < 0 {
		return 0
	}

	return prev
}

func AsCollector(user *auth.User) Collector {
	if user == nil {
		return Collector{}
	}

	return Collector{
		ID: user.ID,
	}
}
