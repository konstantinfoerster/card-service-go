package auth

import (
	"context"
	"fmt"
	"net/http"

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
		validate: func(ctx context.Context, token *JWT, clientID string) (*Claims, error) {
			payload, err := validator.Validate(ctx, token.IDToken, clientID)
			if err != nil {
				return nil, fmt.Errorf("id token invalid %w", err)
			}
			cEmail := payload.Claims["email"]
			cSub := payload.Claims["sub"]

			id, ok := cSub.(string)
			if !ok {
				return nil, fmt.Errorf("claims.sub is not a string but %T", cSub)
			}

			email := ""
			if cEmail != nil {
				var ok bool
				email, ok = cEmail.(string)
				if !ok {
					return nil, fmt.Errorf("claims.email is not a string but %T", cEmail)
				}
			}

			return NewClaims(id, email), nil
		},
	}, nil
}
