package adapter

import (
	"github.com/dustin/articles-backend/internal/article"
	"github.com/dustin/articles-backend/internal/classifier"
	"github.com/dustin/articles-backend/internal/rating"
	"github.com/google/uuid"
)

// ClassifierToMetadataExtractor adapts classifier.Classifier to article.MetadataExtractor
type ClassifierToMetadataExtractor struct {
	classifier classifier.Classifier
}

// NewClassifierToMetadataExtractor creates a new adapter
func NewClassifierToMetadataExtractor(c classifier.Classifier) article.MetadataExtractor {
	return &ClassifierToMetadataExtractor{
		classifier: c,
	}
}

func (a *ClassifierToMetadataExtractor) Extract(url string) (*article.ExtractedMetadata, error) {
	// Call classifier with empty HTML to let it fetch the content
	result, err := a.classifier.Classify(url, "")
	if err != nil {
		return nil, err
	}

	// Convert classifier.Result to article.ExtractedMetadata
	return &article.ExtractedMetadata{
		Title:       result.Title,
		Description: result.Description,
		Content:     result.Content,
		ImageURL:    result.Image,
		WordCount:   result.WordCount,
		Confidence:  result.Confidence,
	}, nil
}

// ArticleServiceToRatingArticleService adapts article.Service to rating.ArticleService
type ArticleServiceToRatingArticleService struct {
	service article.Service
}

// NewArticleServiceToRatingArticleService creates a new adapter
func NewArticleServiceToRatingArticleService(s article.Service) rating.ArticleService {
	return &ArticleServiceToRatingArticleService{
		service: s,
	}
}

func (a *ArticleServiceToRatingArticleService) GetArticle(id uuid.UUID, userID uuid.UUID) (*rating.Article, error) {
	articleEntity, err := a.service.GetArticle(id, userID)
	if err != nil {
		return nil, err
	}

	// Convert article.Article to rating.Article
	return &rating.Article{
		ID:     articleEntity.ID,
		UserID: articleEntity.UserID,
		Title:  articleEntity.Title,
		URL:    articleEntity.URL,
	}, nil
}
