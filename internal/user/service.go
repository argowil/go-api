package user

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Service contains user management business logic.
type Service struct {
	repo Repository
}

// NewService creates a user Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetByID(ctx context.Context, id uint) (*User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]User, error) {
	return s.repo.List(ctx)
}

// Create hashes the password and persists the new user.
func (s *Service) Create(ctx context.Context, req CreateUserRequest) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	role := req.Role
	if role == "" {
		role = "employee"
	}

	u := &User{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         role,
		Active:       true,
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Update applies the non-nil fields from req to the existing user.
// Uses FindByIDAdmin so inactive users can also be edited.
func (s *Service) Update(ctx context.Context, id uint, req UpdateUserRequest) (*User, error) {
	u, err := s.repo.FindByIDAdmin(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.FirstName != nil {
		u.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		u.LastName = *req.LastName
	}
	if req.Email != nil {
		u.Email = *req.Email
	}
	if req.Role != nil {
		u.Role = *req.Role
	}
	if req.Active != nil {
		u.Active = *req.Active
	}

	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Delete soft-deletes a user (sets active = false).
func (s *Service) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}
