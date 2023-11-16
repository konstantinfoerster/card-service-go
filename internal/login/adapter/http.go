package adapter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
	commonhttp "github.com/konstantinfoerster/card-service-go/internal/common/http"
	"github.com/rs/zerolog/log"
)

const stateCookie = "TOKEN_STATE"
const sessionCookie = "SESSION"

// Routes All login and user related routes.
func Routes(app fiber.Router, cfg config.Oidc, svc oidc.Service, timeSvc common.TimeService) {
	authMiddleware := oidc.NewOauthMiddleware(svc, oidc.FromConfig(cfg))

	app.Get("/login/:provider", login(cfg, svc, timeSvc))
	app.Post("/login/:provider/token", exchangeCode(svc, timeSvc))
	app.Post("/logout", logout(svc, timeSvc))
	app.Get("/user", authMiddleware, getCurrentUser())
}

type AuthCode struct {
	Code  string `json:"code" form:"code"`
	State string `json:"state" form:"state"`
}

func login(cfg config.Oidc, svc oidc.Service, timeSvc common.TimeService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		provider, err := requiredParam(c, "provider")
		if err != nil {
			log.Error().Err(err).Msg("provider must not be empty")

			return err
		}

		url, err := svc.GetAuthURL(provider)
		if err != nil {
			return err
		}

		expires := timeSvc.Now().Add(cfg.StateCookieAge)
		setCookie(c, stateCookie, url.State, expires)

		return c.Redirect(url.URL, http.StatusFound)
	}
}

func exchangeCode(svc oidc.Service, timeSvc common.TimeService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		provider, err := requiredParam(c, "provider")
		if err != nil {
			log.Error().Err(err).Msg("provider must not be empty")

			return err
		}

		cookieValue := strings.TrimSpace(c.Cookies(stateCookie))
		if cookieValue == "" {
			return common.NewInvalidInputMsg("code-exchange-missing-state", "missing state")
		}

		clearCookie(c, stateCookie, timeSvc.Now())

		var body AuthCode
		if err = c.BodyParser(&body); err != nil {
			return common.NewInvalidInputError(err, "code-exchange-invalid-body", "invalid body")
		}

		if cookieValue == "" || body.State != cookieValue {
			return common.NewInvalidInputMsg("code-exchange-invalid-state", "invalid state")
		}

		user, token, err := svc.Authenticate(provider, body.Code)
		if err != nil {
			return err
		}

		token64, err := toBase64(token)
		if err != nil {
			return err
		}

		expires := timeSvc.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
		setCookie(c, sessionCookie, token64, expires)

		return commonhttp.RenderJSON(c, contextUserToUser(user))
	}
}

func logout(svc oidc.Service, timeSvc common.TimeService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		clearCookie(c, stateCookie, timeSvc.Now())

		cookieValue := strings.TrimSpace(c.Cookies(sessionCookie))
		if cookieValue == "" {
			return c.SendStatus(http.StatusOK)
		}
		clearCookie(c, sessionCookie, timeSvc.Now())

		token, err := fromBase64(cookieValue)
		if err != nil {
			return err
		}

		provider, err := required(token.Provider, "provider")
		if err != nil {
			return err
		}

		err = svc.Logout(provider, token)
		if err != nil {
			return err
		}

		return c.SendStatus(http.StatusOK)
	}
}

func getCurrentUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		u, err := auth.UserFromCtx(c)
		if err != nil {
			return c.SendStatus(http.StatusForbidden)
		}

		return commonhttp.RenderJSON(c, contextUserToUser(u))
	}
}

func requiredParam(c *fiber.Ctx, name string) (string, error) {
	return required(c.Params(name), name)
}

func required(value, name string) (string, error) {
	if strings.TrimSpace(value) == "" {
		sErr := fmt.Errorf(name + "must not be empty")

		return "", common.NewInvalidInputError(sErr, "required-parameter", sErr.Error())
	}

	return value, nil
}

func toBase64(jwtToken *oidc.JSONWebToken) (string, error) {
	rawJwToken, err := json.Marshal(&jwtToken)
	if err != nil {
		return "", common.NewUnknownError(err, "unable-to-encode-token")
	}

	return base64.URLEncoding.EncodeToString(rawJwToken), nil
}

func fromBase64(base64Token string) (*oidc.JSONWebToken, error) {
	sToken, err := base64.URLEncoding.DecodeString(base64Token)
	if err != nil {
		return nil, common.NewInvalidInputError(err, "unable-to-decode-token", "invalid token encoding")
	}

	var token oidc.JSONWebToken
	if err = json.Unmarshal(sToken, &token); err != nil {
		return nil, common.NewInvalidInputError(err, "unable-to-parse-token", "invalid token format")
	}

	return &token, nil
}

func setCookie(c *fiber.Ctx, name string, value string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		HTTPOnly: true,
		Secure:   true,
		Path:     "/api",
		SameSite: fiber.CookieSameSiteStrictMode,
		Expires:  expires,
	})
}

func clearCookie(c *fiber.Ctx, name string, now time.Time) {
	cookie := c.Cookies(name)
	if cookie == "" {
		return
	}

	minusOneWeek := now.Add(-7 * 24 * time.Hour)
	setCookie(c, name, "invalid", minusOneWeek)
}

func contextUserToUser(u *auth.User) *User {
	username := u.Username
	if username == "" {
		username = "Unknown"
	}

	return &User{
		Username: username,
	}
}

type User struct {
	Username string `json:"username,omitempty"`
}
