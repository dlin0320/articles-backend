package rating

import (
	"errors"
	"fmt"
	"time"

	"github.com/dustin/articles-backend/internal/utils"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/google/uuid"
)

// service implements the Service interface
type service struct {
	repo           Repository
	articleService ArticleService
	logger         *logger.Logger
}

// NewService creates a new rating service
func NewService(repo Repository, articleService ArticleService, log *logger.Logger) Service {
	return &service{
		repo:           repo,
		articleService: articleService,
		logger:         log.WithComponent("rating-service"),
	}
}

func (s *service) RateArticle(userID, articleID uuid.UUID, score int) (*Rating, error) {
	s.logger.Info("Rating article " + articleID.String() + " by user " + userID.String() + " with score " + utils.IntToString(score))

	// Validate score
	if score < 1 || score > 5 {
		s.logger.Error("Invalid rating score " + utils.IntToString(score) + " for article " + articleID.String() + " by user " + userID.String())
		return nil, fmt.Errorf("score must be between 1 and 5, got %d", score)
	}

	// Verify article exists and user ownership
	_, err := s.articleService.GetArticle(articleID, userID)
	if err != nil {
		s.logger.Error("Article not found or access denied " + articleID.String() + " for user " + userID.String() + ": " + err.Error())
		return nil, errors.New("article not found")
	}

	// Check if rating already exists
	_, err = s.repo.FindByUserAndArticle(userID, articleID)
	if err == nil {
		// Rating already exists, update it inline
		existingRating, _ := s.repo.FindByUserAndArticle(userID, articleID)
		existingRating.Score = score
		existingRating.UpdatedAt = time.Now()

		if updateErr := s.repo.Update(existingRating); updateErr != nil {
			s.logger.Error("Failed to update rating for article " + articleID.String() + " by user " + userID.String() + " score " + utils.IntToString(score) + ": " + updateErr.Error())
			return nil, updateErr
		}

		s.logger.Info("Rating updated successfully for article " + articleID.String() + " by user " + userID.String() + " score " + utils.IntToString(score))
		return existingRating, nil
	}

	// Create new rating
	rating := &Rating{
		UserID:    userID,
		ArticleID: articleID,
		Score:     score,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(rating); err != nil {
		s.logger.Error("Failed to create rating for article " + articleID.String() + " by user " + userID.String() + " score " + utils.IntToString(score) + ": " + err.Error())
		return nil, err
	}

	s.logger.Info("Rating created successfully for article " + articleID.String() + " by user " + userID.String() + " score " + utils.IntToString(score))

	return rating, nil
}

func (s *service) GetRating(userID, articleID uuid.UUID) (*Rating, error) {
	rating, err := s.repo.FindByUserAndArticle(userID, articleID)
	if err != nil {
		s.logger.Info("Rating not found for article " + articleID.String() + " by user " + userID.String())
		return nil, errors.New("rating not found")
	}

	return rating, nil
}

func (s *service) DeleteRating(userID, articleID uuid.UUID) error {
	s.logger.Info("Deleting rating for article " + articleID.String() + " by user " + userID.String())

	// Verify rating exists
	_, err := s.repo.FindByUserAndArticle(userID, articleID)
	if err != nil {
		return errors.New("rating not found")
	}

	if err := s.repo.Delete(userID, articleID); err != nil {
		s.logger.Error("Failed to delete rating for article " + articleID.String() + " by user " + userID.String() + ": " + err.Error())
		return err
	}

	s.logger.Info("Rating deleted successfully for article " + articleID.String() + " by user " + userID.String())

	return nil
}
