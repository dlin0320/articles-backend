package user

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system with optimized GORM tags
type User struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null;size:255"`
	PasswordHash string    `json:"-" gorm:"not null;size:255"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Associations - will be loaded explicitly when needed
	Articles []Article `json:"articles,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Ratings  []Rating  `json:"ratings,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// Article represents the article entity (forward declaration for association)
type Article struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
	Title  string
}

// Rating represents the rating entity (forward declaration for association)
type Rating struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	ArticleID uuid.UUID `gorm:"type:uuid;primaryKey"`
	Score     int
}

// Repository defines the interface for user data access
type Repository interface {
	Create(user *User) error
	FindByEmail(email string) (*User, error)
	FindByID(id uuid.UUID) (*User, error)
}

// Service defines the interface for user business logic
type Service interface {
	SignUp(email, password string) (*User, error)
	Login(email, password string) (string, error)
	GetUserByID(id uuid.UUID) (*User, error)
	ValidateToken(tokenString string) (*User, error)
}

// CreateUserRequest represents user creation request
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UserResponse represents user in API responses (without password)
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts User to UserResponse
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// TableName returns the table name for GORM
func (User) TableName() string {
	return "users"
}
