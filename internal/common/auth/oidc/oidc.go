package oidc

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
	"github.com/rs/zerolog/log"
)

func ClaimsToUser(claims *Claims) *auth.User {
	return &auth.User{
		ID:       claims.ID,
		Username: claims.Email,
	}
}

func ExtractClaimsFromCookie(rawCookie string, sp *SupportedProvider) (*Claims, error) {
	jwtToken, err := JwtFromCookie(rawCookie)
	if err != nil {
		return nil, err
	}

	p, err := sp.Find(jwtToken.Provider)
	if err != nil {
		return nil, err
	}
	claims, err := p.ValidateToken(context.Background(), jwtToken)
	if err != nil {
		log.Error().Err(err).Msg("failed to validate jwt")

		return nil, err
	}

	return claims, nil
}

func JwtFromCookie(cookie string) (*JSONWebToken, error) {
	if cookie == "" {
		return nil, common.NewAuthorizationError(fmt.Errorf("no running session found"), "no-session")
	}
	rawJwt, err := base64.URLEncoding.DecodeString(cookie)
	if err != nil {
		log.Error().Err(err).Msg("failed to decode base64 string")

		return nil, common.NewUnknownError(err, "unable-to-decode-token")
	}

	var jwtToken JSONWebToken
	decoder := json.NewDecoder(bytes.NewReader(rawJwt))
	err = decoder.Decode(&jwtToken)
	if err != nil {
		log.Error().Err(err).Msgf("failed to decode jwt %s", rawJwt)

		return nil, common.NewUnknownError(err, "unable-to-decode-token")
	}

	return &jwtToken, nil
}
