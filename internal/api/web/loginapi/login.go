package loginapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
)

const stateCookie = "TOKEN_STATE"

type TimeService interface {
	Now() time.Time
}

type Service interface {
	AuthURL(provider string) (auth.RedirectURL, error)
	Authenticate(ctx context.Context, provider string, code string) (auth.Claims, *auth.JWT, error)
	Logout(ctx context.Context, token *auth.JWT) error
}

// Routes All login and user related routes.
func Routes(app fiber.Router, auth web.AuthMiddleware, cfg auth.Config, svc Service, tSvc TimeService) {
	log := slog.Default()

	app.Get("/login/:provider/callback", exchangeCode(cfg, svc, tSvc))
	app.Get("/login/:provider", login(cfg, svc, tSvc, log))
	app.Get("/logout", logout(cfg, svc, tSvc))
	app.Get("/user", auth.Required(), getCurrentUser())
}

func login(cfg auth.Config, svc Service, timeSvc TimeService, log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		provider, err := requiredParam(c, "provider")
		if err != nil {
			log.Error("provider must not be empty", slog.Any("error", err))

			return err
		}

		url, err := svc.AuthURL(provider)
		if err != nil {
			return err
		}

		expires := timeSvc.Now().Add(cfg.StateCookieAge)
		setStateCookie(c, url.State, expires)

		return c.Redirect(url.URL, http.StatusFound)
	}
}

func exchangeCode(cfg auth.Config, svc Service, timeSvc TimeService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		provider, err := requiredParam(c, "provider")
		if err != nil {
			return err
		}

		cookieValue := strings.TrimSpace(c.Cookies(stateCookie))
		if cookieValue == "" {
			return aerrors.NewInvalidInputMsg("code-exchange-missing-state", "missing state")
		}

		clearCookie(c, stateCookie, timeSvc.Now())

		authErr := c.Params("error")
		if authErr != "" {
			return aerrors.NewInvalidInputMsg("code-exchange-auth-error", authErr)
		}

		rawState, err := requiredQuery(c, "state")
		if err != nil {
			return err
		}

		code, err := requiredQuery(c, "code")
		if err != nil {
			return err
		}

		if cookieValue == "" || rawState != cookieValue {
			return aerrors.NewInvalidInputMsg("code-exchange-invalid-state", "invalid state")
		}

		claims, token, err := svc.Authenticate(c.Context(), provider, code)
		if err != nil {
			return err
		}

		token64, err := token.Encode()
		if err != nil {
			return err
		}

		expires := timeSvc.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
		setCookie(c, cfg.SessionCookieName, token64, expires)

		if web.AcceptsHTML(c) {
			return c.Render("finish_login", nil)
		}

		return web.RenderJSON(c, web.NewClientUser(web.NewUser(claims.ID, claims.Email)))
	}
}

func logout(cfg auth.Config, svc Service, timeSvc TimeService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		clearCookie(c, stateCookie, timeSvc.Now())

		cookieValue := strings.TrimSpace(c.Cookies(cfg.SessionCookieName))
		if cookieValue == "" {
			return c.SendStatus(http.StatusOK)
		}
		clearCookie(c, cfg.SessionCookieName, timeSvc.Now())

		token, err := auth.DecodeSession(cookieValue)
		if err != nil {
			return err
		}

		err = svc.Logout(c.Context(), token)
		if err != nil {
			return err
		}

		if web.AcceptsHTML(c) {
			return c.Redirect("/")
		}

		return c.SendStatus(http.StatusOK)
	}
}

func getCurrentUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		u, err := web.UserFromCtx(c)
		if err != nil {
			return c.SendStatus(http.StatusForbidden)
		}

		return web.RenderJSON(c, web.NewClientUser(u))
	}
}

func requiredParam(c *fiber.Ctx, name string) (string, error) {
	return required(c.Params(name), name)
}

func requiredQuery(c *fiber.Ctx, name string) (string, error) {
	return required(c.Query(name), name)
}

func required(value, name string) (string, error) {
	if strings.TrimSpace(value) == "" {
		msg := fmt.Sprintf("%s must not be empty", name)
		err := fmt.Errorf("%s, %w", msg, web.ErrInvalidField)

		return "", aerrors.NewInvalidInputError(err, "required-parameter", msg)
	}

	return value, nil
}

func setCookie(c *fiber.Ctx, name string, value string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: fiber.CookieSameSiteStrictMode,
		Expires:  expires,
	})
}

func setStateCookie(c *fiber.Ctx, value string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     stateCookie,
		Value:    value,
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: fiber.CookieSameSiteLaxMode,
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
