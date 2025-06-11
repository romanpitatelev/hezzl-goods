package rest

import (
	"context"
	"crypto/rsa"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/romanpitatelev/hezzl-goods/internal/entity"
	"github.com/rs/zerolog/log"
)

const (
	tokenLength    = 3
	authFailedText = "authorization failed"
	tokenDuration  = 24 * time.Hour
)

//go:embed keys/public_key.pem
var publicKeyData []byte

//nolint:funlen
func (s *Server) jwtAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			s.errorUnauthorizedResponse(w, entity.ErrInvalidToken)

			return
		}

		headerParts := strings.Split(header, " ")

		if headerParts[0] != "Bearer" {
			s.errorUnauthorizedResponse(w, entity.ErrInvalidToken)

			return
		}

		encodedToken := strings.Split(headerParts[1], ".")
		if len(encodedToken) != tokenLength {
			s.errorUnauthorizedResponse(w, entity.ErrInvalidToken)

			return
		}

		token, err := jwt.ParseWithClaims(headerParts[1], &entity.Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, entity.ErrInvalidSigningMethod
			}

			return s.key, nil
		})
		if err != nil {
			log.Debug().Err(err).Msg("JWT validation failed")
			s.errorUnauthorizedResponse(w, entity.ErrInvalidToken)

			return
		}

		claims, ok := token.Claims.(*entity.Claims)
		if !ok || !token.Valid {
			s.errorUnauthorizedResponse(w, entity.ErrInvalidToken)

			return
		}

		if claims.ExpiresAt.Before(time.Now()) {
			s.errorUnauthorizedResponse(w, entity.ErrInvalidToken)

			return
		}

		userInfo := entity.User{
			ID:        claims.UserID,
			Email:     *claims.Email,
			Role:      *claims.Role,
			Points:    claims.Points,
			CreatedAt: claims.CreatedAt,
			UpdatedAt: *claims.UpdatedAt,
		}

		r = r.WithContext(context.WithValue(r.Context(), entity.User{}, userInfo))

		next.ServeHTTP(w, r)
	})
}

func NewClaims() *entity.Claims {
	tokenTime := time.Now().Add(tokenDuration)

	return &entity.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(tokenTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
}

func ReadPublicKey() (*rsa.PublicKey, error) {
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyData)
	if err != nil {
		return nil, fmt.Errorf("error parsing public key: %w", err)
	}

	return publicKey, nil
}

func GetPublicKey() *rsa.PublicKey {
	key, err := ReadPublicKey()
	if err != nil {
		return nil
	}

	return key
}

func (s *Server) errorUnauthorizedResponse(w http.ResponseWriter, err error) {
	errResp := fmt.Errorf("%s: %w", authFailedText, err).Error()

	response, err := json.Marshal(errResp)
	if err != nil {
		log.Warn().Err(err).Msg("error marshalling response")
	}

	w.WriteHeader(http.StatusUnauthorized)

	if _, err = w.Write(response); err != nil {
		log.Warn().Err(err).Msg("error writing response")
	}
}
