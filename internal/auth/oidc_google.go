package auth

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
		validate: func(ctx context.Context, token *JSONWebToken, clientID string) (*Claims, error) {
			payload, err := validator.Validate(ctx, token.IDToken, clientID)
			if err != nil {
				return nil, aerrors.NewUnknownError(fmt.Errorf("id token invalid %w", err), "invalid-token")
			}
			cEmail := payload.Claims["email"]
			cSub := payload.Claims["sub"]

			id := cSub.(string)
			email := ""
			if cEmail != nil {
				var ok bool
				email, ok = cEmail.(string)
				if !ok {
					return nil, fmt.Errorf("claims.email is not a string but %T", email)
				}
			}

			return NewClaims(id, email), nil
		},
	}, nil
}
