// Package user provides HTTP handlers, service logic, and repository access
// for the users domain.
package user

import (
	"context"
	"time"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	users *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{users: repo}
}

func (s *Service) Register(ctx context.Context, email, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	usr := &models.User{
		Email:        email,
		PasswordHash: string(hash),
	}

	if err := s.users.Create(ctx, usr); err != nil {
		return nil, err
	}

	return usr, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.users.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.users.Delete(ctx, id)
}

// UpdateMeInput holds the profile fields a user may update.
type UpdateMeInput struct {
	Timezone *string
}

// UpdateMe applies profile updates for the authenticated user.
func (s *Service) UpdateMe(ctx context.Context, id uuid.UUID, input UpdateMeInput) error {
	if input.Timezone != nil {
		if _, err := time.LoadLocation(*input.Timezone); err != nil {
			return ErrInvalidTimezone
		}
		if err := s.users.UpdateTimezone(ctx, id, *input.Timezone); err != nil {
			return err
		}
	}
	return nil
}
