package oidc

import (
	"context"
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/common"
	"google.golang.org/api/idtoken"
)

func DefaultGoogleProviderConfiguration() *Provider {
	return &Provider{
		Name:      "google",
		AuthURL:   "https://accounts.google.com/o/oauth2/auth",
		TokenURL:  "https://accounts.google.com/o/oauth2/token",
		RevokeURL: "https://oauth2.googleapis.com/revoke",
		ClientID:  "",
		Secret:    "",
		Scope:     "openid email",
		Validate: func(ctx context.Context, token *JSONWebToken, clientID string) (*Claims, error) {
			payload, err := idtoken.Validate(ctx, token.IDToken, clientID)
			if err != nil {
				return nil, common.NewUnknownError(fmt.Errorf("id token invalid %w", err), "invalid-token")
			}
			email := payload.Claims["email"]
			sub := payload.Claims["sub"]

			claims := Claims{ID: sub.(string)}
			if email != nil {
				var ok bool
				claims.Email, ok = email.(string)
				if !ok {
					return nil, fmt.Errorf("claims.email is not a string but %T", email)
				}
			}

			return &claims, nil
		},
	}
}
