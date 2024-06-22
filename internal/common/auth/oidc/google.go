package oidc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/konstantinfoerster/card-service-go/internal/common/aerrors"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
)

func googleProvider(client *http.Client) (*provider, error) {
	validator, err := idtoken.NewValidator(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
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
		validate: func(ctx context.Context, token *JSONWebToken, clientID string) (*claims, error) {
			payload, err := validator.Validate(ctx, token.IDToken, clientID)
			if err != nil {
				return nil, aerrors.NewUnknownError(fmt.Errorf("id token invalid %w", err), "invalid-token")
			}
			email := payload.Claims["email"]
			sub := payload.Claims["sub"]

			c := claims{ID: sub.(string)}
			if email != nil {
				var ok bool
				c.Email, ok = email.(string)
				if !ok {
					return nil, fmt.Errorf("claims.email is not a string but %T", email)
				}
			}

			return &c, nil
		},
	}, nil
}
