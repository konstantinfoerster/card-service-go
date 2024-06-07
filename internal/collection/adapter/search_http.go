package adapter

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/collection/application"
	"github.com/konstantinfoerster/card-service-go/internal/collection/domain"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
	commonio "github.com/konstantinfoerster/card-service-go/internal/common/io"
	"github.com/rs/zerolog/log"
)

func SearchRoutes(r fiber.Router, cfg config.Oidc, authSvc oidc.UserService, searchSvc application.SearchService) {
	relaxedAuthMiddleware := oidc.NewOauthMiddleware(authSvc, oidc.FromConfig(cfg), oidc.AllowEmptyCookie())

	r.Get("/cards", relaxedAuthMiddleware, search(searchSvc))
	r.Post("/detect", relaxedAuthMiddleware, detect(searchSvc))
}

func search(svc application.SearchService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// when user is not set, the user specific collection data won't be loaded
		user, _ := auth.UserFromCtx(c)

		searchTerm := c.Query("name")
		page := newPage(c)
		log.Debug().Msgf("search for card with name %s on page %#v", searchTerm, page)
		result, err := svc.Search(c.Context(), collector(user), c.Query("name"), page)
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
					return commonhttp.RenderPartial(c, "search", data)
				}

				return commonhttp.RenderPartial(c, "card_list", data)
			}

			return commonhttp.RenderPage(c, "search", data)
		}

		log.Debug().Msgf("render json page %v, more = %v, size = %d",
			pagedResult.Page, pagedResult.HasMore, len(pagedResult.Data))

		return commonhttp.RenderJSON(c, pagedResult)
	}
}

func detect(svc application.SearchService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// when user is not set, the user specific collection data won't be loaded
		user, _ := auth.UserFromCtx(c)

		fHeader, err := c.FormFile("file")
		if err != nil {
			return common.NewInvalidInputError(err, "invalid-file", "failed to read file from form")
		}

		file, err := fHeader.Open()
		if err != nil {
			return common.NewInvalidInputError(err, "invalid-file", "failed to open file")
		}
		defer commonio.Close(file)

		result, err := svc.Detect(c.Context(), collector(user), file)
		if err != nil {
			return err
		}

		pagedResult := newMatchesResult(result)
		if commonhttp.AcceptsHTML(c) || commonhttp.IsHTMX(c) {
			data := fiber.Map{
				"Page": pagedResult,
			}

			return commonhttp.RenderPartial(c, "search", data)
		}

		return commonhttp.RenderJSON(c, pagedResult)
	}
}

func newPage(c *fiber.Ctx) common.Page {
	size, _ := strconv.Atoi(c.Query("size", ""))
	page, _ := strconv.Atoi(c.Query("page", "0"))

	return common.NewPage(page, size)
}

func newMatchesResult(matches domain.Matches) *PagedResult[Card] {
	data := make([]Card, len(matches))
	for i, m := range matches {
		s := m.Score
		data[i] = Card{
			Item:  newItem(m.ID, m.Amount),
			Name:  m.Name,
			Image: m.Image,
			Score: &s,
		}
	}

	firstPage := 1
	nextPage := 2

	return &PagedResult[Card]{
		Data:     data,
		HasMore:  false,
		Page:     firstPage,
		NextPage: nextPage,
	}
}
func newPagedResult(pr domain.Cards) *PagedResult[Card] {
	data := make([]Card, len(pr.Result))
	for i, c := range pr.Result {
		data[i] = Card{
			Item:  newItem(c.ID, c.Amount),
			Name:  c.Name,
			Image: c.Image,
		}
	}

	return &PagedResult[Card]{
		Data:     data,
		HasMore:  pr.HasMore,
		Page:     pr.Page,
		NextPage: pr.Page + 1,
	}
}

type PagedResult[T any] struct {
	Data     []T  `json:"data"`
	HasMore  bool `json:"hasMore"`
	Page     int  `json:"page"`
	NextPage int  `json:"nextPage"`
}

type Card struct {
	Score *int   `json:"score,omitempty"`
	Name  string `json:"name"`
	Image string `json:"image,omitempty"`
	Item
}

func (c Card) WithScore(v int) Card {
	c.Score = &v

	return c
}

func newItem(id int, amount int) Item {
	return Item{
		ID:     id,
		Amount: amount,
	}
}

type Item struct {
	ID     int `json:"id"`
	Amount int `json:"amount,omitempty"`
}

func (i Item) NextAmount() int {
	return i.Amount + 1
}

func (i Item) PreviousAmount() int {
	prev := i.Amount - 1
	if prev < 0 {
		return 0
	}

	return prev
}

type Query struct {
	Name string
}
