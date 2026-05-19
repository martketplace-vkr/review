package review

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	catalogclient "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/client"
	catalogdomain "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/domain"
	orderclient "github.com/martketplace-vkr/order/pkg/api/grpc/v1/client"
	"github.com/martketplace-vkr/review/domain"
	"github.com/martketplace-vkr/review/internal/service/reviewerrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	repository     repository
	catalog        catalogClient
	order          orderClient
	catalogTimeout time.Duration
	orderTimeout   time.Duration
}

func New(
	repository repository,
	catalog catalogClient,
	order orderClient,
	catalogTimeout time.Duration,
	orderTimeout time.Duration,
) *Service {
	return &Service{
		repository:     repository,
		catalog:        catalog,
		order:          order,
		catalogTimeout: catalogTimeout,
		orderTimeout:   orderTimeout,
	}
}

func (s *Service) ListProductReviews(ctx context.Context, productID int64) ([]domain.Review, *domain.RatingSummary, error) {
	if err := validateID("product_id", productID); err != nil {
		return nil, nil, err
	}

	reviews, err := s.repository.ListProductReviews(ctx, productID)
	if err != nil {
		return nil, nil, mapRepositoryError(err)
	}

	summary, err := s.repository.RatingSummary(ctx, productID)
	if err != nil {
		return nil, nil, mapRepositoryError(err)
	}

	return reviews, summary, nil
}

func (s *Service) CreateReview(ctx context.Context, req CreateReviewRequest) (*domain.Review, *domain.RatingSummary, error) {
	if err := validateID("product_id", req.ProductID); err != nil {
		return nil, nil, err
	}
	if err := validateID("author_user_id", req.AuthorUserID); err != nil {
		return nil, nil, err
	}
	if req.Rating < 1 || req.Rating > 5 {
		return nil, nil, fmt.Errorf("%w: rating must be between 1 and 5", reviewerrors.ErrInvalidArgument)
	}

	comment := strings.TrimSpace(req.Comment)
	if comment == "" {
		return nil, nil, fmt.Errorf("%w: comment is required", reviewerrors.ErrInvalidArgument)
	}

	product, err := s.getProduct(ctx, req.ProductID)
	if err != nil {
		return nil, nil, err
	}
	if product.GetVendorId() <= 0 {
		return nil, nil, fmt.Errorf("%w: product vendor_id is required", reviewerrors.ErrInvalidArgument)
	}

	hasOrder, vendorID, err := s.hasSuccessfulOrder(ctx, req.AuthorUserID, req.ProductID)
	if err != nil {
		return nil, nil, err
	}
	if !hasOrder {
		return nil, nil, reviewerrors.ErrNoPurchase
	}
	if vendorID > 0 && vendorID != product.GetVendorId() {
		return nil, nil, reviewerrors.ErrNoPurchase
	}

	authorName := strings.TrimSpace(req.AuthorName)
	if authorName == "" {
		authorName = "Покупатель"
	}

	review, err := s.repository.CreateReview(ctx, domain.Review{
		ProductID:    req.ProductID,
		VendorID:     product.GetVendorId(),
		AuthorUserID: req.AuthorUserID,
		AuthorName:   authorName,
		Rating:       req.Rating,
		Comment:      comment,
	}, normalizeImageURLs(req.ImageURLs))
	if err != nil {
		return nil, nil, mapRepositoryError(err)
	}

	summary, err := s.repository.RatingSummary(ctx, req.ProductID)
	if err != nil {
		return nil, nil, mapRepositoryError(err)
	}

	return review, summary, nil
}

func (s *Service) VoteReview(ctx context.Context, reviewID int64, userID int64, vote string) (*domain.Review, error) {
	if err := validateID("review_id", reviewID); err != nil {
		return nil, err
	}
	if err := validateID("user_id", userID); err != nil {
		return nil, err
	}

	vote = strings.TrimSpace(vote)
	if vote != domain.VoteHelpful && vote != domain.VoteNotHelpful {
		return nil, fmt.Errorf("%w: unknown vote", reviewerrors.ErrInvalidArgument)
	}

	review, err := s.repository.UpsertVote(ctx, reviewID, userID, vote)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return review, nil
}

func (s *Service) ReportReview(ctx context.Context, reviewID int64, reporterUserID int64, reason string, details string) (*domain.Report, error) {
	if err := validateID("review_id", reviewID); err != nil {
		return nil, err
	}
	if err := validateID("reporter_user_id", reporterUserID); err != nil {
		return nil, err
	}

	reason = strings.TrimSpace(reason)
	if !isValidReportReason(reason) {
		return nil, fmt.Errorf("%w: unknown report reason", reviewerrors.ErrInvalidArgument)
	}

	if _, err := s.repository.GetReview(ctx, reviewID); err != nil {
		return nil, mapRepositoryError(err)
	}

	report, err := s.repository.CreateReport(ctx, domain.Report{
		ReviewID:       reviewID,
		ReporterUserID: reporterUserID,
		Reason:         reason,
		Details:        strings.TrimSpace(details),
	})
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return report, nil
}

func (s *Service) ListVendorReviews(ctx context.Context, vendorID int64) ([]domain.Review, error) {
	if err := validateID("vendor_id", vendorID); err != nil {
		return nil, err
	}

	reviews, err := s.repository.ListVendorReviews(ctx, vendorID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return reviews, nil
}

func (s *Service) ReplyReview(ctx context.Context, vendorID int64, reviewID int64, comment string) (*domain.Review, error) {
	if err := validateID("vendor_id", vendorID); err != nil {
		return nil, err
	}
	if err := validateID("review_id", reviewID); err != nil {
		return nil, err
	}

	comment = strings.TrimSpace(comment)
	if comment == "" {
		return nil, fmt.Errorf("%w: comment is required", reviewerrors.ErrInvalidArgument)
	}

	review, err := s.repository.UpsertReply(ctx, reviewID, vendorID, comment)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return review, nil
}

func (s *Service) DeleteReviewReply(ctx context.Context, vendorID int64, reviewID int64) (*domain.Review, error) {
	if err := validateID("vendor_id", vendorID); err != nil {
		return nil, err
	}
	if err := validateID("review_id", reviewID); err != nil {
		return nil, err
	}

	review, err := s.repository.DeleteReply(ctx, reviewID, vendorID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return review, nil
}

func (s *Service) DisputeReview(ctx context.Context, vendorID int64, reviewID int64, reason string) (*domain.Review, error) {
	if err := validateID("vendor_id", vendorID); err != nil {
		return nil, err
	}
	if err := validateID("review_id", reviewID); err != nil {
		return nil, err
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "Несправедливый отзыв"
	}

	review, err := s.repository.CreateDispute(ctx, reviewID, vendorID, reason)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return review, nil
}

func (s *Service) CancelReviewDispute(ctx context.Context, vendorID int64, reviewID int64) (*domain.Review, error) {
	if err := validateID("vendor_id", vendorID); err != nil {
		return nil, err
	}
	if err := validateID("review_id", reviewID); err != nil {
		return nil, err
	}

	review, err := s.repository.CancelDispute(ctx, reviewID, vendorID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return review, nil
}

func (s *Service) ListReviewDisputes(ctx context.Context, status string) ([]domain.Review, error) {
	reviews, err := s.repository.ListDisputedReviews(ctx, strings.TrimSpace(status))
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return reviews, nil
}

func (s *Service) ListReviewReports(ctx context.Context, status string) ([]domain.ReportedReview, error) {
	reports, err := s.repository.ListReportedReviews(ctx, strings.TrimSpace(status))
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return reports, nil
}

func (s *Service) ResolveReviewDispute(ctx context.Context, adminID int64, reviewID int64, disputeID int64, decision string, comment string) (*domain.Review, error) {
	if err := validateID("admin_id", adminID); err != nil {
		return nil, err
	}
	if err := validateID("review_id", reviewID); err != nil {
		return nil, err
	}
	if err := validateID("dispute_id", disputeID); err != nil {
		return nil, err
	}

	decision = strings.TrimSpace(decision)
	if decision != domain.DisputeAccepted && decision != domain.DisputeRejected {
		return nil, fmt.Errorf("%w: decision must be accepted or rejected", reviewerrors.ErrInvalidArgument)
	}

	review, err := s.repository.ResolveDispute(ctx, reviewID, disputeID, adminID, decision == domain.DisputeAccepted, strings.TrimSpace(comment))
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return review, nil
}

func (s *Service) DeleteReview(ctx context.Context, adminID int64, reviewID int64) error {
	if err := validateID("admin_id", adminID); err != nil {
		return err
	}
	if err := validateID("review_id", reviewID); err != nil {
		return err
	}

	if err := s.repository.DeleteReview(ctx, reviewID, adminID); err != nil {
		return mapRepositoryError(err)
	}

	return nil
}

func (s *Service) getProduct(ctx context.Context, productID int64) (*catalogdomain.Product, error) {
	timeout := s.catalogTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := s.catalog.GetProduct(callCtx, &catalogclient.GetProductRequest{ProductId: productID})
	if err != nil {
		return nil, err
	}
	if resp.GetProduct() == nil {
		return nil, reviewerrors.ErrReviewNotFound
	}

	return resp.GetProduct(), nil
}

func (s *Service) hasSuccessfulOrder(ctx context.Context, userID int64, productID int64) (bool, int64, error) {
	timeout := s.orderTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := s.order.HasSuccessfulProductOrder(callCtx, &orderclient.HasSuccessfulProductOrderRequest{
		UserId:    userID,
		ProductId: productID,
	})
	if err != nil {
		return false, 0, err
	}

	return resp.GetHasOrder(), resp.GetVendorId(), nil
}

type CreateReviewRequest struct {
	ProductID    int64
	AuthorUserID int64
	AuthorName   string
	Rating       int32
	Comment      string
	ImageURLs    []string
}

func validateID(field string, value int64) error {
	if value <= 0 {
		return fmt.Errorf("%w: %s must be greater than zero", reviewerrors.ErrInvalidArgument, field)
	}

	return nil
}

func normalizeImageURLs(imageURLs []string) []string {
	result := make([]string, 0, len(imageURLs))
	for _, imageURL := range imageURLs {
		imageURL = strings.TrimSpace(imageURL)
		if imageURL != "" {
			result = append(result, imageURL)
		}
	}

	return result
}

func isValidReportReason(reason string) bool {
	switch reason {
	case domain.ReportContent, domain.ReportMedia, domain.ReportSpam:
		return true
	default:
		return false
	}
}

func mapRepositoryError(err error) error {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return reviewerrors.ErrReviewNotFound
	case status.Code(err) == codes.NotFound:
		return reviewerrors.ErrReviewNotFound
	default:
		return err
	}
}
