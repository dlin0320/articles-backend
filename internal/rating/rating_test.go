package rating

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRating(t *testing.T) {
	t.Run("Create new rating", func(t *testing.T) {
		userID := uuid.New()
		articleID := uuid.New()
		rating := Rating{
			UserID:    userID,
			ArticleID: articleID,
			Score:     5,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.Equal(t, userID, rating.UserID)
		assert.Equal(t, articleID, rating.ArticleID)
		assert.Equal(t, 5, rating.Score)
		assert.True(t, rating.IsValidScore())
		assert.NotZero(t, rating.CreatedAt)
		assert.NotZero(t, rating.UpdatedAt)
	})

	t.Run("IsValidScore", func(t *testing.T) {
		testCases := []struct {
			name     string
			score    int
			expected bool
		}{
			{"Valid score 1", 1, true},
			{"Valid score 3", 3, true},
			{"Valid score 5", 5, true},
			{"Invalid score 0", 0, false},
			{"Invalid score 6", 6, false},
			{"Invalid negative score", -1, false},
			{"Invalid high score", 100, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rating := Rating{
					UserID:    uuid.New(),
					ArticleID: uuid.New(),
					Score:     tc.score,
				}
				assert.Equal(t, tc.expected, rating.IsValidScore())
			})
		}
	})

	t.Run("ToResponse", func(t *testing.T) {
		userID := uuid.New()
		articleID := uuid.New()
		now := time.Now()

		rating := Rating{
			UserID:    userID,
			ArticleID: articleID,
			Score:     4,
			CreatedAt: now,
			UpdatedAt: now,
		}

		response := rating.ToResponse()

		assert.Equal(t, rating.UserID, response.UserID)
		assert.Equal(t, rating.ArticleID, response.ArticleID)
		assert.Equal(t, rating.Score, response.Score)
		assert.Equal(t, rating.CreatedAt, response.CreatedAt)
		assert.Equal(t, rating.UpdatedAt, response.UpdatedAt)
	})

	t.Run("Update existing rating", func(t *testing.T) {
		rating := Rating{
			UserID:    uuid.New(),
			ArticleID: uuid.New(),
			Score:     3,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		originalUpdatedAt := rating.UpdatedAt
		time.Sleep(10 * time.Millisecond)

		rating.Score = 5
		rating.UpdatedAt = time.Now()

		assert.Equal(t, 5, rating.Score)
		assert.True(t, rating.UpdatedAt.After(originalUpdatedAt))
		assert.True(t, rating.IsValidScore())
	})

	t.Run("Table name", func(t *testing.T) {
		rating := Rating{}
		assert.Equal(t, "ratings", rating.TableName())
	})
}

func TestRateArticleRequest(t *testing.T) {
	t.Run("Valid request", func(t *testing.T) {
		req := RateArticleRequest{
			Score: 4,
		}

		assert.Equal(t, 4, req.Score)
		assert.True(t, req.Score >= 1 && req.Score <= 5)
	})
}
