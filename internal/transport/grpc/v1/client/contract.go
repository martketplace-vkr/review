package client

import (
	"context"

	"github.com/martketplace-vkr/review/domain"
	"github.com/martketplace-vkr/review/internal/service/review"
)

type service interface {
	ListProductReviews(ctx context.Context, productID int64) ([]domain.Review, *domain.RatingSummary, error)
	CreateReview(ctx context.Context, req review.CreateReviewRequest) (*domain.Review, *domain.RatingSummary, error)
	VoteReview(ctx context.Context, reviewID int64, userID int64, vote string) (*domain.Review, error)
	ReportReview(ctx context.Context, reviewID int64, reporterUserID int64, reason string, details string) (*domain.Report, error)
}
