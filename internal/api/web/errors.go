package web

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/rs/zerolog/log"
)

const ContentType = "application/problem+json"

type ProblemJSON struct {
	// A URI reference [RFC3986] that identifies the problem type.
	// This specification encourages that, when dereferenced, it provides human-readable documentation for the
	// problem type.
	Type string `json:"type,omitempty"`
	// A short, human-readable summary of the problem type.
	Title string `json:"title"`
	Key   string `json:"key"`
	// The HTTP status code [RFC7231].
	Status int `json:"status"`
	// A human-readable explanation specific to this occurrence of the problem.
	Detail string `json:"detail,omitempty"`
	// A URI reference that identifies the specific occurrence of the problem.
	Instance string `json:"instance,omitempty"`
}

func NewProblemJSON(title string, key string, status int) *ProblemJSON {
	return &ProblemJSON{
		Title:  title,
		Key:    key,
		Status: status,
	}
}

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
	statusCode := StatusUnauthorized
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
