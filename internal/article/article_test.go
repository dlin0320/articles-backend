package article

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestArticle(t *testing.T) {
	t.Run("Create new article", func(t *testing.T) {
		userID := uuid.New()
		article := Article{
			ID:              uuid.New(),
			UserID:          userID,
			URL:             "https://example.com/article",
			Title:           "Test Article",
			Description:     "Test Description",
			ImageURL:        "https://example.com/image.jpg",
			MetadataStatus:  MetadataStatusSuccess,
			RetryCount:      0,
			ConfidenceScore: 0.85,
			ClassifierUsed:  "readability",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		assert.NotEqual(t, uuid.Nil, article.ID)
		assert.Equal(t, userID, article.UserID)
		assert.Equal(t, "https://example.com/article", article.URL)
		assert.Equal(t, "Test Article", article.Title)
		assert.Equal(t, "Test Description", article.Description)
		assert.Equal(t, "https://example.com/image.jpg", article.ImageURL)
		assert.Equal(t, MetadataStatusSuccess, article.MetadataStatus)
		assert.Equal(t, 0, article.RetryCount)
		assert.Equal(t, 0.85, article.ConfidenceScore)
		assert.Equal(t, "readability", article.ClassifierUsed)
		assert.False(t, article.CreatedAt.IsZero())
		assert.False(t, article.UpdatedAt.IsZero())
	})

	t.Run("Metadata status constants", func(t *testing.T) {
		assert.Equal(t, "pending", MetadataStatusPending)
		assert.Equal(t, "success", MetadataStatusSuccess)
		assert.Equal(t, "failed", MetadataStatusFailed)
	})

	t.Run("IsOwnedBy", func(t *testing.T) {
		userID := uuid.New()
		otherUserID := uuid.New()

		article := Article{
			ID:     uuid.New(),
			UserID: userID,
		}

		assert.True(t, article.IsOwnedBy(userID))
		assert.False(t, article.IsOwnedBy(otherUserID))
	})

	t.Run("NeedsMetadataExtraction", func(t *testing.T) {
		testCases := []struct {
			name       string
			status     string
			retryCount int
			expected   bool
		}{
			{"Pending status", MetadataStatusPending, 0, true},
			{"Failed with low retry", MetadataStatusFailed, 1, true},
			{"Failed with high retry", MetadataStatusFailed, 5, false},
			{"Success status", MetadataStatusSuccess, 0, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				article := Article{
					MetadataStatus: tc.status,
					RetryCount:     tc.retryCount,
				}
				assert.Equal(t, tc.expected, article.NeedsMetadataExtraction())
			})
		}
	})

	t.Run("ToResponse", func(t *testing.T) {
		article := Article{
			ID:              uuid.New(),
			UserID:          uuid.New(),
			URL:             "https://example.com/article",
			Title:           "Test Article",
			Description:     "Test Description",
			ImageURL:        "https://example.com/image.jpg",
			WordCount:       500,
			MetadataStatus:  MetadataStatusSuccess,
			ConfidenceScore: 0.9,
			ClassifierUsed:  "readability",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		response := article.ToResponse()

		assert.Equal(t, article.ID, response.ID)
		assert.Equal(t, article.UserID, response.UserID)
		assert.Equal(t, article.URL, response.URL)
		assert.Equal(t, article.Title, response.Title)
		assert.Equal(t, article.Description, response.Description)
		assert.Equal(t, article.ImageURL, response.ImageURL)
		assert.Equal(t, article.WordCount, response.WordCount)
		assert.Equal(t, article.MetadataStatus, response.MetadataStatus)
		assert.Equal(t, article.ConfidenceScore, response.ConfidenceScore)
		assert.Equal(t, article.ClassifierUsed, response.ClassifierUsed)
	})

	t.Run("ToResponse with ratings", func(t *testing.T) {
		article := Article{
			ID:     uuid.New(),
			UserID: uuid.New(),
			Title:  "Test Article",
			Ratings: []Rating{
				{Score: 5},
				{Score: 4},
				{Score: 5},
			},
		}

		response := article.ToResponse()

		assert.NotNil(t, response.AverageRating)
		assert.NotNil(t, response.RatingCount)
		assert.Equal(t, float64(14)/float64(3), *response.AverageRating) // (5+4+5)/3
		assert.Equal(t, 3, *response.RatingCount)
	})

	t.Run("Table name", func(t *testing.T) {
		article := Article{}
		assert.Equal(t, "articles", article.TableName())
	})
}

func TestBuildPaginationResponse(t *testing.T) {
	articles := []*Article{
		{
			ID:    uuid.New(),
			Title: "Article 1",
		},
		{
			ID:    uuid.New(),
			Title: "Article 2",
		},
	}

	response := BuildPaginationResponse(articles, 10, 1, 5)

	assert.Len(t, response.Articles, 2)
	assert.Equal(t, int64(10), response.Total)
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 5, response.Limit)
	assert.Equal(t, 2, response.Pages) // 10/5 = 2 pages
}
