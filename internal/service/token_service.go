package service

import (
	"auth-service/internal/model"
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

type TokenService struct {
	repo       model.TokenRepository
	jwtSecret  []byte
	webhookURL string
}

func NewTokenService(repo model.TokenRepository, jwtSecret, webhookURL string) *TokenService {
	return &TokenService{
		repo:       repo,
		jwtSecret:  []byte(jwtSecret),
		webhookURL: webhookURL,
	}
}

func (s *TokenService) GenerateAccessToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *TokenService) GenerateRefreshToken(userID, userAgent, ip string) (string, error) {
	raw := fmt.Sprintf("%s:%d", userID, time.Now().UnixNano())
	hash := sha512.Sum512([]byte(raw))
	token := base64.StdEncoding.EncodeToString(hash[:])

	hashed, err := bcrypt.GenerateFromPassword(hash[:], bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	err = s.repo.StoreRefreshToken(userID, string(hashed), userAgent, ip)
	return token, err
}

func (s *TokenService) RefreshTokens(userID, refreshToken, userAgent, ip string) (string, string, error) {
	stored, err := s.repo.FindActiveToken(userID)
	if err != nil || stored.Used {
		return "", "", errors.New("refresh token invalid or used")
	}

	decoded, err := base64.StdEncoding.DecodeString(refreshToken)
	if err != nil {
		return "", "", errors.New("invalid refresh token encoding")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(stored.TokenHash), decoded); err != nil {
		_ = s.repo.InvalidateToken(userID)
		return "", "", errors.New("invalid refresh token")
	}

	if stored.UserAgent != userAgent {
		_ = s.repo.InvalidateToken(userID)
		return "", "", errors.New("user-agent mismatch")
	}

	if stored.IPAddress != ip {
		go s.sendWebhook(userID, ip, userAgent)
	}

	_ = s.repo.InvalidateToken(userID)

	accessToken, err := s.GenerateAccessToken(userID)
	if err != nil {
		return "", "", err
	}
	refreshTokenNew, err := s.GenerateRefreshToken(userID, userAgent, ip)
	return accessToken, refreshTokenNew, err
}

func (s *TokenService) InvalidateTokens(userID string) error {
	return s.repo.InvalidateToken(userID)
}

func (s *TokenService) ParseAccessToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS512.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid access token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", errors.New("user_id not found")
	}
	return userID, nil
}

func (s *TokenService) sendWebhook(userID, ip, userAgent string) {
	payload := fmt.Sprintf(`{"user_id": "%s", "ip": "%s", "user_agent": "%s"}`, userID, ip, userAgent)
	http.Post(s.webhookURL, "application/json", bytes.NewBuffer([]byte(payload)))
}
