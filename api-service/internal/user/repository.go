package user

import (
	"context"
	"errors"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, usr *models.User) error {
	result := r.db.WithContext(ctx).Create(usr)
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return ErrEmailTaken
		}
		return result.Error
	}
	return nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var usr models.User
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&usr)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}
	return &usr, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var usr models.User
	result := r.db.WithContext(ctx).First(&usr, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}
	return &usr, nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}

// UpdateTimezone sets the IANA timezone string for the given user.
func (r *Repository) UpdateTimezone(ctx context.Context, id uuid.UUID, timezone string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", id).
		Update("timezone", timezone).Error
}
