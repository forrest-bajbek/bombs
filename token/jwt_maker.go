package token

import (
	"fmt"
	"os"

	"github.com/forrest-bajbek/bombs/utils"
	"github.com/golang-jwt/jwt/v5"
)

type JWTMaker struct {
	secretKey string
}

func NewJWTMaker() *JWTMaker {
	secretKey := os.Getenv("BOMBS_JWT_ENCRYPTION_KEY")
	if len(secretKey) == 0 {
		secretKey = utils.GenerateRandomHexToken(64)
	} else if len(secretKey) < 64 {
		panic("If you specify BOMBS_JWT_ENCRYPTION_KEY, it must be at least 64 characters.")
	}
	return &JWTMaker{secretKey: secretKey}
}

// Invite Token
// ------------------------------------------------------------------------------------
func (maker *JWTMaker) CreateInviteToken(username string) (string, *InviteClaims, error) {
	claims := NewInviteTokenClaims(username)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(maker.secretKey))
	if err != nil {
		return "", nil, fmt.Errorf("error signing token: %w", err)
	}
	return tokenStr, claims, nil
}

func (maker *JWTMaker) ValidateInviteToken(tokenStr string) (*InviteClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &InviteClaims{}, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, fmt.Errorf("invalid token singing method")
		}
		return []byte(maker.secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("error parsing token: %w", err)
	}

	claims, ok := token.Claims.(*InviteClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// Auth Token
// ------------------------------------------------------------------------------------
func (maker *JWTMaker) CreateAuthToken(userID int) (string, *AuthClaims, error) {
	claims := NewAuthTokenClaims(userID)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(maker.secretKey))
	if err != nil {
		return "", nil, fmt.Errorf("error signing token: %w", err)
	}
	return tokenStr, claims, nil
}

func (maker *JWTMaker) ValidateAuthToken(tokenStr string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, fmt.Errorf("invalid token singing method")
		}
		return []byte(maker.secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("error parsing token: %w", err)
	}

	claims, ok := token.Claims.(*AuthClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
