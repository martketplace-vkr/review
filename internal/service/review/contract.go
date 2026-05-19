package review

import (
	"context"

	catalogclient "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/client"
	orderclient "github.com/martketplace-vkr/order/pkg/api/grpc/v1/client"
	"github.com/martketplace-vkr/review/domain"
	"google.golang.org/grpc"
)

type repository interface {
	CreateReview(ctx context.Context, review domain.Review, imageURLs []string) (*domain.Review, error)
	ListProductReviews(ctx context.Context, productID int64) ([]domain.Review, error)
	ListVendorReviews(ctx context.Context, vendorID int64) ([]domain.Review, error)
	ListDisputedReviews(ctx context.Context, status string) ([]domain.Review, error)
	ListReportedReviews(ctx context.Context, status string) ([]domain.ReportedReview, error)
	GetReview(ctx context.Context, reviewID int64) (*domain.Review, error)
	UpsertVote(ctx context.Context, reviewID int64, userID int64, vote string) (*domain.Review, error)
	CreateReport(ctx context.Context, report domain.Report) (*domain.Report, error)
	UpsertReply(ctx context.Context, reviewID int64, vendorID int64, comment string) (*domain.Review, error)
	DeleteReply(ctx context.Context, reviewID int64, vendorID int64) (*domain.Review, error)
	CreateDispute(ctx context.Context, reviewID int64, vendorID int64, reason string) (*domain.Review, error)
	CancelDispute(ctx context.Context, reviewID int64, vendorID int64) (*domain.Review, error)
	ResolveDispute(ctx context.Context, reviewID int64, disputeID int64, adminID int64, accepted bool, comment string) (*domain.Review, error)
	DeleteReview(ctx context.Context, reviewID int64, adminID int64) error
	RatingSummary(ctx context.Context, productID int64) (*domain.RatingSummary, error)
}

type catalogClient interface {
	GetProduct(ctx context.Context, in *catalogclient.GetProductRequest, opts ...grpc.CallOption) (*catalogclient.GetProductResponse, error)
}

type orderClient interface {
	HasSuccessfulProductOrder(ctx context.Context, in *orderclient.HasSuccessfulProductOrderRequest, opts ...grpc.CallOption) (*orderclient.HasSuccessfulProductOrderResponse, error)
}
