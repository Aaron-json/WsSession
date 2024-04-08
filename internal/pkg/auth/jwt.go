package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
)

type TokenClaims struct {
	UserId string `json:"userId"`
	jwt.StandardClaims
}

// will be embedded in the request context
type UserAuth struct {
	UserId string
}

// used to set and get the keys of the request context values.
type authKey string

const AUTH_CONTEXT_KEY authKey = "auth"

type Token int

const (
	AccessToken  Token = 0
	RefreshToken Token = 1
)

func (c authKey) String() string {
	return string(c)
}

func NewToken(userId string, tokenType Token) (string, error) {
	claims := TokenClaims{
		UserId: userId,
	}
	var expiryHrs int64
	if tokenType == AccessToken {
		expiryHrs = 24
	} else if tokenType == RefreshToken {
		expiryHrs = 48
	} else {
		return "", errors.New("invalid token type")
	}
	claims.ExpiresAt = time.Now().Add(time.Hour * time.Duration(expiryHrs)).Unix()
	// add the expiry embedded field
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(os.Getenv("jwt_secret"))
}

func parseAcessToken(r *http.Request) (*http.Request, error) {

	_, userToken, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found {
		return nil, errors.New("invalid token format")
	}
	claims := TokenClaims{}
	_, err := jwt.ParseWithClaims(userToken, &claims, func(t *jwt.Token) (interface{}, error) {
		key, ok := os.LookupEnv("jwt_secret")
		if !ok {
			return nil, errors.New("JWT secret not found")
		}
		return key, nil
	})
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(r.Context(), AUTH_CONTEXT_KEY, &UserAuth{
		UserId: claims.UserId,
	})

	newReq := r.WithContext(ctx)
	return newReq, nil
}

// middleware to handle user authentication data in the request
func ParseAcessToken(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nr, err := parseAcessToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, nr)
	})
}

func GetAuth(r *http.Request) (*UserAuth, error) {
	val := r.Context().Value(AUTH_CONTEXT_KEY)
	ctxVal, ok := val.(*UserAuth)
	if !ok {
		return nil, errors.New("invalid context value")
	}
	return ctxVal, nil
}
