package web

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const MethodGet = http.MethodGet
const MethodPost = http.MethodPost

const StatusOK = http.StatusOK
const StatusFound = http.StatusFound
const StatusUnauthorized = http.StatusUnauthorized
const StatusBadRequest = http.StatusBadRequest
const StatusInternalServerError = http.StatusInternalServerError

const HeaderHTMXRequest = "HX-Request"

// IsHTMX true if the request is a HTMX request, false otherwise.
func IsHTMX(c *fiber.Ctx) bool {
	return strings.ToLower(c.Get(HeaderHTMXRequest)) == "true"
}

// AcceptsHTML true if the request expectects HTML as response, false otherwise.
func AcceptsHTML(c *fiber.Ctx) bool {
	return strings.Contains(c.Get(fiber.HeaderAccept), fiber.MIMETextHTML)
}
