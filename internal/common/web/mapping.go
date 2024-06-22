package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/rs/zerolog/log"
)

// FIXME: This should not be part of web
func RespondWithProblemJSON(c *fiber.Ctx, err error) error {
	var appErr aerrors.AppError
	if !errors.As(err, &appErr) {
		return internalError(c, "internal-error", err)
	}

	switch appErr.ErrorType {
	case aerrors.ErrTypeInvalidInput:
		return badRequest(c, appErr.Msg, appErr.Key, err)
	case aerrors.ErrTypeAuthorization:
		return unauthorized(c, appErr.Key, err)
	default:
		return internalError(c, appErr.Key, err)
	}
}

func unauthorized(c *fiber.Ctx, key string, err error) error {
	statusCode := http.StatusUnauthorized
	log.Error().Err(err).Int("statusCode", statusCode).Str("key", key).Send()

	return c.Status(statusCode).
		JSON(NewProblemJSON(http.StatusText(statusCode), key, statusCode), ContentType)
}

func badRequest(c *fiber.Ctx, title string, key string, err error) error {
	statusCode := http.StatusBadRequest
	log.Error().Err(err).Int("statusCode", statusCode).Str("key", key).Send()

	return c.Status(statusCode).
		JSON(NewProblemJSON(title, key, statusCode), ContentType)
}

func internalError(c *fiber.Ctx, key string, err error) error {
	statusCode := http.StatusInternalServerError
	log.Error().Err(err).Int("statusCode", statusCode).Str("key", key).Send()

	return c.Status(statusCode).
		JSON(NewProblemJSON(http.StatusText(statusCode), key, statusCode), ContentType)
}

func NewClientUser(u *auth.User) *ClientUser {
	if u == nil {
		return nil
	}

	username := u.Username
	if username == "" {
		username = "Unknown"
	}

	initials := []rune(username)[0:2]

	return &ClientUser{
		Username: username,
		Initials: string(initials),
	}
}

type ClientUser struct {
	Username string `json:"username"`
	Initials string `json:"initials"`
}

func NewPage(c *fiber.Ctx) common.Page {
	size, _ := strconv.Atoi(c.Query("size", ""))
	page, _ := strconv.Atoi(c.Query("page", "0"))

	return common.NewPage(page, size)
}

type PagedResponse[T any] struct {
	Data     []T  `json:"data"`
	HasMore  bool `json:"hasMore"`
	Page     int  `json:"page"`
	NextPage int  `json:"nextPage"`
}

