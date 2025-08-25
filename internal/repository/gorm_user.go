package repository

import (
	"fmt"

	userPkg "github.com/dustin/articles-backend/internal/user"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// gormUserRepository implements the user.Repository interface with GORM optimizations
type gormUserRepository struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewGORMUserRepository creates a new GORM-based user repository
func NewGORMUserRepository(db *gorm.DB, log *logger.Logger) userPkg.Repository {
	return &gormUserRepository{
		db:     db,
		logger: log.WithComponent("gorm-user-repository"),
	}
}

func (r *gormUserRepository) Create(user *userPkg.User) error {
	r.logger.Info("Creating user " + user.ID.String() + " with email " + user.Email)

	if err := r.db.Create(user).Error; err != nil {
		r.logger.Error("Failed to create user " + user.ID.String() + " with email " + user.Email + ": " + err.Error())
		return fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.Info("User created successfully: " + user.ID.String() + " with email " + user.Email)

	return nil
}

func (r *gormUserRepository) FindByEmail(email string) (*userPkg.User, error) {
	var user userPkg.User

	// Use index-optimized query with explicit column
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Info("User not found by email: " + email)
			return nil, fmt.Errorf("user not found")
		}

		r.logger.Error("Database error finding user by email " + email + ": " + err.Error())
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &user, nil
}

func (r *gormUserRepository) FindByID(id uuid.UUID) (*userPkg.User, error) {
	var user userPkg.User

	// Use primary key lookup for optimal performance
	err := r.db.First(&user, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Info("User not found by ID: " + id.String())
			return nil, fmt.Errorf("user not found")
		}

		r.logger.Error("Database error finding user by ID " + id.String() + ": " + err.Error())
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &user, nil
}
