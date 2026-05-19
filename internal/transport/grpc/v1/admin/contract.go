package admin

import (
	"context"

	"github.com/martketplace-vkr/review/domain"
)

type service interface {
	ListReviewDisputes(ctx context.Context, status string) ([]domain.Review, error)
	ResolveReviewDispute(ctx context.Context, adminID int64, reviewID int64, disputeID int64, decision string, comment string) (*domain.Review, error)
	DeleteReview(ctx context.Context, adminID int64, reviewID int64) error
}
