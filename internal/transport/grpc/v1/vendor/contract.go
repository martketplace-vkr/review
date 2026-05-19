package vendor

import (
	"context"

	"github.com/martketplace-vkr/review/domain"
)

type service interface {
	ListVendorReviews(ctx context.Context, vendorID int64) ([]domain.Review, error)
	ReplyReview(ctx context.Context, vendorID int64, reviewID int64, comment string) (*domain.Review, error)
	DisputeReview(ctx context.Context, vendorID int64, reviewID int64, reason string) (*domain.Review, error)
}
