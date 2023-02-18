package problemjson

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/rs/zerolog/log"
)

const problemJSONContentType = "application/problem+json"

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

func RespondWithProblemJSON(err error, c *fiber.Ctx) error {
	var appErr *common.AppError
	if !errors.As(err, &appErr) {
		return internalError("internal-error", err, c)
	}

	switch appErr.ErrorType {
	case common.ErrTypeInvalidInput:
		return badRequest(appErr.Msg, appErr.Key, appErr, c)
	case common.ErrTypeAuthorization:
		return unauthorized(appErr.Key, appErr, c)
	default:
		return internalError(appErr.Key, appErr, c)
	}
}

func unauthorized(key string, err error, c *fiber.Ctx) error {
	statusCode := http.StatusUnauthorized
	log.Error().Err(err).Int("statusCode", statusCode).Str("key", key).Send()

	return c.Status(statusCode).
		Type(problemJSONContentType).
		JSON(NewProblemJSON(http.StatusText(statusCode), key, statusCode))
}

func badRequest(title string, key string, err error, c *fiber.Ctx) error {
	statusCode := http.StatusBadRequest
	log.Error().Err(err).Int("statusCode", statusCode).Str("key", key).Send()

	return c.Status(statusCode).
		Type(problemJSONContentType).
		JSON(NewProblemJSON(title, key, statusCode))
}

func internalError(key string, err error, c *fiber.Ctx) error {
	statusCode := http.StatusInternalServerError
	log.Error().Err(err).Int("statusCode", statusCode).Str("key", key).Send()

	return c.Status(statusCode).
		Type(problemJSONContentType).
		JSON(NewProblemJSON(http.StatusText(statusCode), key, statusCode))
}
