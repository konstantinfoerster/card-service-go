package loginapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/konstantinfoerster/card-service-go/internal/aerrors"
	"github.com/konstantinfoerster/card-service-go/internal/api/web"
	"github.com/konstantinfoerster/card-service-go/internal/auth"
)

var (
	errValueRequired        = errors.New("value must not be empty")
	errInvalidValueEncoding = errors.New("invalid value encoding")
	errInvalidValueContent  = errors.New("invalid value content")
	errInvalidValue         = errors.New("invalid value")
)

type Service interface {
	AuthURL(provider string) (string, auth.State, error)
	Authenticate(ctx context.Context, provider, code, verifier string) (*auth.JWT, auth.User, error)
	Revoke(ctx context.Context, token *auth.JWT) error
}

// Routes All login and user related routes.
func Routes(app fiber.Router, store *session.Store,
	auth web.AuthMiddleware, cfg auth.Config, svc Service) {
	app.Get("/login/:provider/callback", exchangeCode(cfg, store, svc))
	app.Get("/login/:provider", login(cfg, svc))
	app.Post("/logout", logout(cfg, store))
	app.Get("/user", auth.Required(), getCurrentUser())
}

func login(cfg auth.Config, svc Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		provider, err := requiredParam(c, "provider")
		if err != nil {
			return err
		}

		url, state, err := svc.AuthURL(provider)
		if err != nil {
			return err
		}

		cValue, err := encodeBase64(state)
		if err != nil {
			return aerrors.NewUnknownError(err, "login-invalid-state")
		}

		stateCookie := buildCookie(cValue, cfg.State)
		c.Cookie(stateCookie)

		return c.Redirect(url, http.StatusFound)
	}
}

func exchangeCode(cfg auth.Config, store *session.Store, svc Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		provider, err := requiredParam(c, "provider")
		if err != nil {
			return err
		}

		cookieValue := strings.TrimSpace(c.Cookies(cfg.State.Name))
		if cookieValue == "" {
			return aerrors.NewInvalidInputMsg("code-exchange-missing-state", "missing state")
		}

		expireCookie(c, cfg.State)

		state, err := decodeBase64(cookieValue)
		if err != nil {
			return aerrors.NewInvalidInputError(err, "code-exchange-invalid-state-cookie", "invalid state")
		}

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

		if rawState != state.Value {
			return aerrors.NewInvalidInputMsg("code-exchange-invalid-state", "invalid state")
		}

		token, user, err := svc.Authenticate(c.Context(), provider, code, state.Verifier)
		if err != nil {
			return err
		}

		// expire the token, we will only need the session cookie to
		// authenticate the user in followup requests
		if err = svc.Revoke(c.Context(), token); err != nil {
			slog.Warn("failed to revoke token", slog.Any("error", err))
		}

		sess, err := store.Get(c)
		if err != nil {
			return aerrors.NewUnknownError(err, "session-create")
		}

		sess.Set(auth.UserContextKey, user)
		if err := sess.Save(); err != nil {
			if dErr := sess.Destroy(); dErr != nil {
				slog.Warn("failed to destroy session after save error",
					slog.Any("save-error", err),
					slog.Any("destroy-error", dErr),
				)
			}

			return aerrors.NewUnknownError(err, "session-save")
		}

		if web.AcceptsHTML(c) {
			return c.Render("finish_login", nil)
		}

		return web.RenderJSON(c, web.NewClientUser(user))
	}
}

func logout(cfg auth.Config, store *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			expireCookie(c, cfg.State)
		}()

		sendOk := func(c *fiber.Ctx) error {
			if web.AcceptsJSON(c) {
				return c.SendStatus(http.StatusOK)
			}

			if web.IsHTMX(c) {
				c.Set(web.HeaderHTMXRefresh, "/")

				return c.SendStatus(http.StatusOK)
			}

			return c.Redirect("/")
		}

		s, err := store.Get(c)
		if err != nil {
			return sendOk(c)
		}

		if err := s.Destroy(); err != nil {
			slog.Warn("failed to destroy session", slog.Any("error", err))

			return sendOk(c)
		}

		return sendOk(c)
	}
}

func getCurrentUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		u, err := auth.UserFromCtx(c)
		if err != nil {
			return c.SendStatus(http.StatusUnauthorized)
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

func buildCookie(value string, cfg auth.Cookie) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     cfg.Name,
		Value:    value,
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		HTTPOnly: true,
		Secure:   true,
		SameSite: cfg.SameSite,
		MaxAge:   int(cfg.ExpiresIn.Seconds()),
	}
}

func expireCookie(c *fiber.Ctx, cfg auth.Cookie) {
	cookieRaw := c.Cookies(cfg.Name)
	if cookieRaw == "" {
		return
	}

	cookie := buildCookie("", cfg)
	cookie.MaxAge = 0
	cookie.Expires = time.Unix(0, 0).UTC()

	c.Cookie(cookie)
}

func encodeBase64(value auth.State) (string, error) {
	rawJSON, err := json.Marshal(&value)
	if err != nil {
		return "", errInvalidValue
	}

	return base64.URLEncoding.EncodeToString(rawJSON), nil
}

func decodeBase64(value string) (auth.State, error) {
	if strings.TrimSpace(value) == "" {
		return auth.State{}, errValueRequired
	}

	rawJSON, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return auth.State{}, errInvalidValueEncoding
	}

	var state auth.State
	if err := json.Unmarshal(rawJSON, &state); err != nil {
		return auth.State{}, errors.Join(err, errInvalidValueContent)
	}

	return state, nil
}
