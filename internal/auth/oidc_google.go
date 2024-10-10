package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
)

var (
	errEmptyToken     = errors.New("empty token")
	errValidateGoogle = errors.New("validate google provider")
)

func googleProvider(client *http.Client) (*provider, error) {
	validator, err := idtoken.NewValidator(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create validator due to %w", err)
	}

	return &provider{
		name:      "google",
		authURL:   "https://accounts.google.com/o/oauth2/auth",
		tokenURL:  "https://accounts.google.com/o/oauth2/token",
		revokeURL: "https://oauth2.googleapis.com/revoke",
		client:    client,
		clientID:  "",
		secret:    "",
		scope:     "openid email",
		validate: func(ctx context.Context, token *JWT, clientID string) (Claims, error) {
			if token == nil {
				return Claims{}, errEmptyToken
			}
			payload, err := validator.Validate(ctx, token.IDToken, clientID)
			if err != nil {
				return Claims{}, fmt.Errorf("id token validation failed with %w", err)
			}
			cEmail := payload.Claims["email"]
			cSub := payload.Claims["sub"]

			id, ok := cSub.(string)
			if !ok {
				return Claims{}, fmt.Errorf("claims.sub is not a string but %T, %w", cSub, errValidateGoogle)
			}

			email := ""
			if cEmail != nil {
				var ok bool
				email, ok = cEmail.(string)
				if !ok {
					return Claims{}, fmt.Errorf("claims.email is not a string but %T, %w", cEmail, errValidateGoogle)
				}
			}

			return NewClaims(id, email), nil
		},
	}, nil
}
