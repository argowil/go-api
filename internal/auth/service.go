package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"argowil/backend/internal/user"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost is the work factor used for hashing passwords across the application.
// Raise this to increase security at the cost of login latency.
const bcryptCost = bcrypt.DefaultCost

// LoginRequest holds the credentials the client sends.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenPair is returned after a successful login or token refresh.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Service handles authentication business logic.
type Service struct {
	users              user.Repository
	jwtSecret          string
	accessTokenTTL     int
	refreshTokenTTL    int
}

// NewService creates an auth Service.
func NewService(users user.Repository, secret string, accessTTL, refreshTTL int) *Service {
	return &Service{
		users:           users,
		jwtSecret:       secret,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// Login validates credentials and returns a fresh token pair on success.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*TokenPair, error) {
	u, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	return s.issueTokenPair(u)
}

// HashPassword returns a bcrypt hash suitable for storage.
// Use this everywhere a password needs to be hashed so the cost constant stays central.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// ChangePassword hashes the new password, saves it and clears the must_change_password flag.
func (s *Service) ChangePassword(ctx context.Context, userID uint, newPassword string) error {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	u.PasswordHash = hash
	u.MustChangePassword = false
	return s.users.Update(ctx, u)
}

// UserByID exposes user lookup for the handler layer.
func (s *Service) UserByID(ctx context.Context, id uint) (*user.User, error) {
	return s.users.FindByID(ctx, id)
}

// Refresh validates a refresh token and issues a new token pair.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := ParseAccessToken(s.jwtSecret, refreshToken)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}
	u, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return s.issueTokenPair(u)
}

func (s *Service) issueTokenPair(u *user.User) (*TokenPair, error) {
	expiresAt := time.Now().Add(time.Duration(s.accessTokenTTL) * time.Minute)

	access, err := NewAccessToken(s.jwtSecret, u.ID, Role(u.Role), u.FullName(), s.accessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	// Refresh token is a longer-lived JWT with minimal claims.
	// In a production setup you would also persist it so it can be revoked.
	refresh, err := NewAccessToken(s.jwtSecret, u.ID, Role(u.Role), u.FullName(), s.refreshTokenTTL*24*60)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    expiresAt,
	}, nil
}
