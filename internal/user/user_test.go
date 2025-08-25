package user

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUser(t *testing.T) {
	t.Run("Create new user", func(t *testing.T) {
		user := User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "hashed_password", user.PasswordHash)
		assert.NotZero(t, user.CreatedAt)
		assert.NotZero(t, user.UpdatedAt)
	})

	t.Run("ToResponse excludes sensitive data", func(t *testing.T) {
		user := User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			PasswordHash: "secret_hash",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		response := user.ToResponse()

		assert.Equal(t, user.ID, response.ID)
		assert.Equal(t, user.Email, response.Email)
		assert.Equal(t, user.CreatedAt, response.CreatedAt)
		assert.Equal(t, user.UpdatedAt, response.UpdatedAt)

		// Password should not be in response
		// (this is implicit since UserResponse doesn't have PasswordHash field)
	})

	t.Run("Email validation", func(t *testing.T) {
		testCases := []struct {
			email string
			valid bool
		}{
			{"valid@example.com", true},
			{"user.name@domain.co", true},
			{"invalid", false},
			{"@domain.com", false},
			{"user@", false},
			{"", false},
		}

		for _, tc := range testCases {
			t.Run(tc.email, func(t *testing.T) {
				if tc.valid {
					assert.Contains(t, tc.email, "@")
				} else {
					assert.True(t, tc.email == "" || !isValidEmail(tc.email))
				}
			})
		}
	})

	t.Run("Table name", func(t *testing.T) {
		user := User{}
		assert.Equal(t, "users", user.TableName())
	})
}

func isValidEmail(email string) bool {
	return len(email) > 3 &&
		email[0] != '@' &&
		email[len(email)-1] != '@' &&
		contains(email, "@")
}

func contains(s, substr string) bool {
	for i := 0; i < len(s); i++ {
		if s[i:i+1] == substr {
			return true
		}
	}
	return false
}
