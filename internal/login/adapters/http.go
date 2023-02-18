package adapters

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/oidc"
	"github.com/konstantinfoerster/card-service-go/internal/common/problemjson"
	"github.com/konstantinfoerster/card-service-go/internal/config"
	"github.com/rs/zerolog/log"
)

const stateCookie = "TOKEN_STATE"
const sessionCookie = "SESSION"

type AuthCode struct {
	Code  string `json:"code" form:"code"`
	State string `json:"state" form:"state"`
}

func Login(cfg config.Oidc) fiber.Handler {
	return func(c *fiber.Ctx) error {
		p, err := oidc.FindProvider(c.Params("provider"))
		if err != nil {
			return problemjson.RespondWithProblemJSON(err, c)
		}

		state := uuid.New().String()
		maxAgeSeconds := 60
		setCookie(c, stateCookie, state, maxAgeSeconds)

		if err = c.Redirect(p.GetAuthURL(state, cfg.RedirectURI), http.StatusFound); err != nil {
			return problemjson.RespondWithProblemJSON(
				fmt.Errorf("failed to redirect to authorization server, %w", err), c)
		}

		return nil
	}
}

func ExchangeCode(cfg config.Oidc) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cookie := c.Cookies(stateCookie)
		defer clearCookie(c, stateCookie)

		var body AuthCode
		err := c.BodyParser(&body)
		if err != nil {
			return err
		}

		if cookie == "" || body.State != cookie {
			log.Error().Msgf("Found state %s but expected %s", body.State, cookie)
			sErr := fmt.Errorf("invalid state")
			err = common.NewInvalidInputError(sErr, "code-exchange-invalid-state", sErr.Error())

			return problemjson.RespondWithProblemJSON(err, c)
		}

		defer clearCookie(c, stateCookie)

		p, err := oidc.FindProvider(c.Params("provider"))
		if err != nil {
			return problemjson.RespondWithProblemJSON(err, c)
		}

		claims, jwtToken, err := p.ExchangeCode(context.Background(), body.Code, cfg.RedirectURI)
		if err != nil {
			log.Error().Err(err).Msg("failed to exchange code")

			return problemjson.RespondWithProblemJSON(err, c)
		}
		token64, err := toBase64(jwtToken)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal jwt")

			return problemjson.RespondWithProblemJSON(err, c)
		}

		expiresMultiplicator := 2
		setCookie(c, sessionCookie, token64, jwtToken.ExpiresIn*expiresMultiplicator)

		if err = c.JSON(claimsToUser(claims)); err != nil {
			return problemjson.RespondWithProblemJSON(fmt.Errorf("failed to encode claims, %w", err), c)
		}

		return nil
	}
}

func toBase64(jwtToken *oidc.JSONWebToken) (string, error) {
	rawJwToken, err := json.Marshal(&jwtToken)
	if err != nil {
		return "", common.NewUnknownError(err, "unable-to-encode-token")
	}

	return base64.URLEncoding.EncodeToString(rawJwToken), nil
}

func setCookie(c *fiber.Ctx, name string, value string, ageSeconds int) {
	expire := time.Now().Add(time.Second * time.Duration(ageSeconds))
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		HTTPOnly: true,
		MaxAge:   ageSeconds,
		Expires:  expire,
		Secure:   true,
		SameSite: "strict",
	})
}

func clearCookie(c *fiber.Ctx, name string) {
	oneDay := 86400 // one day
	setCookie(c, name, "", -oneDay)
}

func Logout() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer clearCookie(c, sessionCookie)
		defer clearCookie(c, stateCookie)

		jwtToken, err := oidc.JwtFromCookie(c.Cookies(sessionCookie))
		if err != nil {
			return c.SendStatus(http.StatusOK)
		}

		p, err := oidc.FindProvider(jwtToken.Provider)
		if err != nil {
			return c.SendStatus(http.StatusOK)
		}

		err = p.RevokeToken(context.Background(), jwtToken.AccessToken)
		if err != nil {
			log.Error().Err(err).Msg("token revoke failed")

			return c.SendStatus(http.StatusOK)
		}

		return c.SendStatus(http.StatusOK)
	}
}

func GetUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := oidc.ExtractClaimsFromCookie(c.Cookies(sessionCookie))
		if err != nil {
			return c.JSON(&User{
				Username:      "Unknown",
				Authenticated: false,
			})
		}

		return c.JSON(claimsToUser(claims))
	}
}

func claimsToUser(claims *oidc.Claims) *User {
	username := claims.Email
	if username == "" {
		username = "Unknown"
	}

	return &User{
		Username:      username,
		Authenticated: true,
	}
}

type User struct {
	Username      string `json:"username"`
	Authenticated bool   `json:"authenticated"`
}
