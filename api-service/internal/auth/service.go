// Package auth provides login, token rotation, logout, and the JWT middleware
// used to gate protected api-service routes. auth imports user — never the
// reverse.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/DiegoJohnson25/climate-control/api-service/internal/user"
	"github.com/DiegoJohnson25/climate-control/api-service/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	users      *user.Repository
	tokens     *Repository
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewService(users *user.Repository, tokens *Repository, secret string, accessTTLMinutes, refreshTTLDays int) *Service {
	return &Service{
		users:      users,
		tokens:     tokens,
		secret:     []byte(secret),
		accessTTL:  time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL: time.Duration(refreshTTLDays) * 24 * time.Hour,
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error) {
	usr, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", err
	}

	if bcrypt.CompareHashAndPassword([]byte(usr.PasswordHash), []byte(password)) != nil {
		return "", "", ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, usr.ID)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (newAccessToken, newRefreshToken string, err error) {
	userID, err := s.tokens.GetUserID(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return "", "", ErrInvalidToken
		}
		return "", "", err
	}

	if err := s.tokens.Delete(ctx, refreshToken); err != nil && !errors.Is(err, ErrTokenNotFound) {
		return "", "", fmt.Errorf("invalidate old token: %w", err)
	}

	return s.issueTokenPair(ctx, userID)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	err := s.tokens.Delete(ctx, refreshToken)
	if errors.Is(err, ErrTokenNotFound) {
		return nil
	}
	return err
}

func (s *Service) issueTokenPair(ctx context.Context, userID uuid.UUID) (accessToken, refreshToken string, err error) {
	accessToken, err = s.newAccessToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	refreshToken, err = s.newRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	if err := s.tokens.Store(ctx, refreshToken, userID); err != nil {
		return "", "", fmt.Errorf("store refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *Service) newAccessToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(s.accessTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.secret)
}

func (s *Service) newRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *Service) ValidateAccessToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil || !token.Valid {
		return uuid.UUID{}, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.UUID{}, ErrInvalidToken
	}

	idStr, ok := claims["user_id"].(string)
	if !ok {
		return uuid.UUID{}, ErrInvalidToken
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.UUID{}, ErrInvalidToken
	}

	return id, nil
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.users.GetByID(ctx, id)
}
