package user

import (
	"context"

	"github.com/DiegoJohnson25/climate-control/shared/models"
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
