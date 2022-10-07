package httperr

import (
	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service/internal/common"
	"github.com/rs/zerolog/log"
	"net/http"
)

const problemJsonContentType = "application/problem+json"

type ProblemJson struct {
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

func NewProblemJson(title string, key string, status int) *ProblemJson {
	return &ProblemJson{
		Title:  title,
		Key:    key,
		Status: status,
	}
}

func RespondWithProblemJson(err error, c *fiber.Ctx) error {
	appError, ok := err.(common.AppError)
	if !ok {
		return internalError("internal-error", appError, c)
	}
	switch appError.ErrorType {
	default:
		return internalError(appError.Key, appError, c)
	}
}

func internalError(key string, err error, c *fiber.Ctx) error {
	log.Error().Err(err).Str("key", key).Send()
	return c.Status(http.StatusInternalServerError).
		Type(problemJsonContentType).
		JSON(NewProblemJson(http.StatusText(http.StatusInternalServerError), key, http.StatusInternalServerError))
}
