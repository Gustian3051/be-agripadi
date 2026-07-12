package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/domain"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error)
	UpdateProfile(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).
		Create(user).
		Error
}

func (r *userRepository) GetByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	var user domain.User

	err := r.db.WithContext(ctx).
		Where("id = ?", userID).
		First(&user).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User

	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error) {
	var user domain.User

	err := r.db.WithContext(ctx).
		Where("phone_number = ?", phoneNumber).
		First(&user).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *userRepository) UpdateProfile(ctx context.Context, user *domain.User) error {
	result := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", user.ID).
		Updates(map[string]any{
			"name":         user.Name,
			"email":        user.Email,
			"phone_number": user.PhoneNumber,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error {
	result := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Update("password", hashedPassword)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
