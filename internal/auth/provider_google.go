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

func googleProvider(client *http.Client) (OIDCProvider, error) {
	validator, err := idtoken.NewValidator(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return OIDCProvider{}, fmt.Errorf("failed to create validator due to %w", err)
	}

	return OIDCProvider{
		Name:      "google",
		AuthURL:   "https://accounts.google.com/o/oauth2/auth",
		TokenURL:  "https://accounts.google.com/o/oauth2/token",
		RevokeURL: "https://oauth2.googleapis.com/revoke",
		Client:    client,
		ClientID:  "",
		Secret:    "",
		Scopes:    []string{"openid", "email"},
		Validate: func(ctx context.Context, token *JWT, clientID string) (Claim, error) {
			if token == nil {
				return Claim{}, errEmptyToken
			}
			payload, err := validator.Validate(ctx, token.IDToken, clientID)
			if err != nil {
				return Claim{}, fmt.Errorf("id token validation failed with %w", err)
			}
			cEmail := payload.Claims["email"]
			cSub := payload.Claims["sub"]

			id, ok := cSub.(string)
			if !ok {
				return Claim{}, fmt.Errorf("claims.sub is not a string but %T, %w", cSub, errValidateGoogle)
			}

			email := ""
			if cEmail != nil {
				var ok bool
				email, ok = cEmail.(string)
				if !ok {
					return Claim{}, fmt.Errorf("claims.email is not a string but %T, %w", cEmail, errValidateGoogle)
				}
			}

			return NewClaim(id, email), nil
		},
	}, nil
}
