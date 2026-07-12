package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gustian305/backend/internal/domain"
	authdto "github.com/gustian305/backend/internal/dto/auth"
	"github.com/gustian305/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = 24 * time.Hour

type Service struct {
	userRepo  repository.UserRepository
	jwtSecret string
}

func NewService(userRepo repository.UserRepository, jwtSecret string) *Service {
	return &Service{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (s *Service) Register(ctx context.Context, req authdto.RegisterRequest) (*authdto.AuthResponse, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	phoneNumber := normalizePhoneNumber(req.PhoneNumber)

	if name == "" {
		return nil, errors.New("name is required")
	}

	if email == "" {
		return nil, errors.New("email is required")
	}

	if phoneNumber == "" {
		return nil, errors.New("phone number is required")
	}

	if len(req.Password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	if req.Password != req.ConfirmPassword {
		return nil, errors.New("password and confirmation password do not match")
	}

	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if existingUser != nil {
		return nil, errors.New("email already exists")
	}

	existingUser, err = s.userRepo.GetByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		return nil, err
	}

	if existingUser != nil {
		return nil, errors.New("phone number already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &domain.User{
		ID:          uuid.New(),
		Name:        name,
		Email:       email,
		PhoneNumber: phoneNumber,
		Password:    string(hashedPassword),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.buildAuthResponse(user)
}

func (s *Service) Login(ctx context.Context, req authdto.LoginRequest) (*authdto.AuthResponse, error) {
	phoneNumber := normalizePhoneNumber(req.PhoneNumber)
	if phoneNumber == "" {
		return nil, errors.New("phone number is required")
	}

	user, err := s.userRepo.GetByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("invalid phone number or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid phone number or password")
	}

	return s.buildAuthResponse(user)
}

func (s *Service) buildAuthResponse(user *domain.User) (*authdto.AuthResponse, error) {
	expiresAt := time.Now().Add(tokenTTL)
	token, err := s.generateToken(user.ID, expiresAt)
	if err != nil {
		return nil, err
	}

	return &authdto.AuthResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresAt: expiresAt,
		User: authdto.UserResponse{
			ID:          user.ID,
			Name:        user.Name,
			Email:       user.Email,
			PhoneNumber: user.PhoneNumber,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		},
	}, nil
}

func (s *Service) generateToken(userID uuid.UUID, expiresAt time.Time) (string, error) {
	if strings.TrimSpace(s.jwtSecret) == "" {
		return "", errors.New("jwt secret is not configured")
	}

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	claims := map[string]any{
		"user_id": userID.String(),
		"sub":     userID.String(),
		"exp":     expiresAt.Unix(),
		"iat":     time.Now().Unix(),
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	claimBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimBytes)
	signingInput := encodedHeader + "." + encodedClaims

	mac := hmac.New(sha256.New, []byte(s.jwtSecret))
	_, _ = mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

func normalizePhoneNumber(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), "")
}
