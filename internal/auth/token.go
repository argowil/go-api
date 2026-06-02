// Package auth handles password hashing, JWT creation/validation, and HTTP middleware.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Role defines what a user is allowed to do within the platform.
type Role string

const (
	RoleEmployee   Role = "employee"
	RoleTeamLeader Role = "teamleader"
	RoleAdmin      Role = "admin"
)

// Claims is embedded in every access token.
type Claims struct {
	UserID uint   `json:"uid"`
	Role   Role   `json:"role"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

// NewAccessToken mints a signed JWT that expires after ttl minutes.
func NewAccessToken(secret string, userID uint, role Role, name string, ttlMinutes int) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		Name:   name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(ttlMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ParseAccessToken validates the signature and expiry of a JWT and returns its claims.
func ParseAccessToken(secret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
