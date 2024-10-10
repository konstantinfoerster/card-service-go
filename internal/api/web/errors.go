package web

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidField = errors.New("invalid field value")
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
	var e *fiber.Error
	if errors.As(err, &e) {
		return sendError(c, e.Code, "api-error", e.Message, e)
	}

	var appErr aerrors.AppError
	if !errors.As(err, &appErr) {
		return sendError(c, StatusInternalServerError, "internal-error", "", err)
	}

	var code int
	switch appErr.ErrorType {
	case aerrors.ErrTypeInvalidInput:
		code = StatusBadRequest
	case aerrors.ErrTypeAuthorization:
		code = StatusUnauthorized
	default:
		code = StatusInternalServerError
	}

	return sendError(c, code, appErr.Key, appErr.Msg, err)
}

func sendError(c *fiber.Ctx, code int, key, title string, err error) error {
	if title == "" {
		title = http.StatusText(code)
	}
	log.Error().Err(err).Int("code", code).Str("key", key).Str("title", title).Send()

	return c.Status(code).
		JSON(NewProblemJSON(title, key, code), ContentType)
}
