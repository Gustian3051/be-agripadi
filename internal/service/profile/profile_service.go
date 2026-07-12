package profile

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/domain"
	profiledto "github.com/gustian305/backend/internal/dto/profile"
	"github.com/gustian305/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	userRepo repository.UserRepository
}

func NewService(userRepo repository.UserRepository) *Service {
	return &Service{
		userRepo: userRepo,
	}
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*profiledto.Response, error) {
	user, err := s.getUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return toProfileResponse(user), nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req profiledto.UpdateRequest) (*profiledto.Response, error) {
	user, err := s.getUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.New("name is required")
		}
		user.Name = name
	}

	if req.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*req.Email))
		if email == "" {
			return nil, errors.New("email is required")
		}

		existingUser, err := s.userRepo.GetByEmail(ctx, email)
		if err != nil {
			return nil, err
		}

		if existingUser != nil && existingUser.ID != user.ID {
			return nil, errors.New("email already exists")
		}

		user.Email = email
	}

	if req.PhoneNumber != nil {
		phoneNumber := strings.Join(strings.Fields(strings.TrimSpace(*req.PhoneNumber)), "")
		if phoneNumber == "" {
			return nil, errors.New("phone number is required")
		}

		existingUser, err := s.userRepo.GetByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			return nil, err
		}

		if existingUser != nil && existingUser.ID != user.ID {
			return nil, errors.New("phone number already exists")
		}

		user.PhoneNumber = phoneNumber
	}

	if err := s.userRepo.UpdateProfile(ctx, user); err != nil {
		return nil, err
	}

	updatedUser, err := s.getUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return toProfileResponse(updatedUser), nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, req profiledto.ChangePasswordRequest) error {
	user, err := s.getUser(ctx, userID)
	if err != nil {
		return err
	}

	if len(req.NewPassword) < 8 {
		return errors.New("new password must be at least 8 characters")
	}

	if req.NewPassword != req.ConfirmPassword {
		return errors.New("new password and confirm password do not match")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.NewPassword)) == nil {
		return errors.New("new password must be different from current password")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(ctx, user.ID, string(hashedPassword))
}

func (s *Service) getUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	if userID == uuid.Nil {
		return nil, errors.New("user id is required")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	return user, nil
}

func toProfileResponse(user *domain.User) *profiledto.Response {
	if user == nil {
		return nil
	}

	return &profiledto.Response{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}
