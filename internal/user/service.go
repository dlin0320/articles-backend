package user

import (
	"errors"
	"fmt"
	"time"

	"github.com/dustin/articles-backend/config"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// service implements the Service interface
type service struct {
	repo      Repository
	jwtSecret string
	jwtExpiry time.Duration
	logger    *logger.Logger
}

// NewService creates a user service with JWT validation and defaults
func NewService(cfg *config.JWTConfig, repo Repository, log *logger.Logger) (*service, error) {
	// Set defaults for nil or empty config values
	secret := "change-me-in-production"
	if cfg != nil && cfg.Secret != "" {
		secret = cfg.Secret
	}

	var expiry time.Duration = 24 * time.Hour
	if cfg != nil && cfg.Expiration != "" {
		duration, err := time.ParseDuration(cfg.Expiration)
		if err != nil {
			return nil, fmt.Errorf("invalid JWT expiration '%s': %v", cfg.Expiration, err)
		}
		expiry = duration
	}

	return &service{
		repo:      repo,
		jwtSecret: secret,
		jwtExpiry: expiry,
		logger:    log.WithComponent("user-service"),
	}, nil
}

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (s *service) SignUp(email, password string) (*User, error) {
	s.logger.Info("User signup attempt for email: " + email)

	// Check if user exists
	existing, _ := s.repo.FindByEmail(email)
	if existing != nil {
		s.logger.Info("Signup failed - user already exists: " + email)
		return nil, errors.New("user already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash password for " + email + ": " + err.Error())
		return nil, err
	}

	// Create user
	user := &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = s.repo.Create(user)
	if err != nil {
		s.logger.Error("Failed to create user " + email + ": " + err.Error())
		return nil, err
	}

	s.logger.Info("User created successfully: " + email + " (ID: " + user.ID.String() + ")")

	return user, nil
}

func (s *service) Login(email, password string) (string, error) {
	s.logger.Info("User login attempt for email: " + email)

	// Find user
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		s.logger.Info("Login failed - user not found: " + email)
		return "", errors.New("invalid credentials")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		s.logger.Info("Login failed - invalid password for " + email + " (ID: " + user.ID.String() + ")")
		return "", errors.New("invalid credentials")
	}

	// Generate JWT token
	token, err := s.generateToken(user)
	if err != nil {
		s.logger.Error("Failed to generate JWT token for " + email + " (ID: " + user.ID.String() + "): " + err.Error())
		return "", err
	}

	s.logger.Info("User logged in successfully: " + email + " (ID: " + user.ID.String() + ")")

	return token, nil
}

func (s *service) GetUserByID(id uuid.UUID) (*User, error) {
	return s.repo.FindByID(id)
}

func (s *service) ValidateToken(tokenString string) (*User, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	// Check if token is valid
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Parse user ID
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, errors.New("invalid user ID in token")
	}

	// Get user from database
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return user, nil
}

func (s *service) generateToken(user *User) (string, error) {
	// Create claims
	claims := Claims{
		UserID: user.ID.String(),
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "articles-backend",
			Subject:   user.ID.String(),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
