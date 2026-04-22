package token

import (
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type InviteClaims struct {
	Username string
	jwt.RegisteredClaims
}

func NewInviteTokenClaims(username string) *InviteClaims {
	return &InviteClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "bombs",
			Subject:   username,
			Audience:  jwt.ClaimStrings{"invite"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now().Add(time.Duration(3) * time.Second)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(3) * time.Minute)),
		},
	}
}

type AuthClaims struct {
	UserID string
	jwt.RegisteredClaims
}

func NewAuthTokenClaims(userID int) *AuthClaims {
	return &AuthClaims{
		UserID: strconv.Itoa(userID),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "bombs",
			Subject:   strconv.Itoa(userID),
			Audience:  jwt.ClaimStrings{"auth"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(15) * time.Minute)),
		},
	}
}
