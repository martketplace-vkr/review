package client

import (
	"context"
	"errors"

	"github.com/martketplace-vkr/review/internal/service/review"
	"github.com/martketplace-vkr/review/internal/service/reviewerrors"
	"github.com/martketplace-vkr/review/internal/transport/grpc/v1/mapper"
	clientpb "github.com/martketplace-vkr/review/pkg/api/grpc/v1/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	service service
	clientpb.UnimplementedReviewClientServiceServer
}

func New(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListProductReviews(ctx context.Context, req *clientpb.ListProductReviewsRequest) (*clientpb.ListProductReviewsResponse, error) {
	reviews, summary, err := h.service.ListProductReviews(ctx, req.GetProductId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &clientpb.ListProductReviewsResponse{
		Reviews: mapper.ReviewsToProto(reviews),
		Summary: mapper.RatingSummaryToProto(summary),
	}, nil
}

func (h *Handler) CreateReview(ctx context.Context, req *clientpb.CreateReviewRequest) (*clientpb.CreateReviewResponse, error) {
	created, summary, err := h.service.CreateReview(ctx, review.CreateReviewRequest{
		ProductID:    req.GetProductId(),
		AuthorUserID: req.GetAuthorUserId(),
		AuthorName:   req.GetAuthorName(),
		Rating:       req.GetRating(),
		Comment:      req.GetComment(),
		ImageURLs:    req.GetImageUrls(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &clientpb.CreateReviewResponse{
		Review:  mapper.ReviewToProto(created),
		Summary: mapper.RatingSummaryToProto(summary),
	}, nil
}

func (h *Handler) VoteReview(ctx context.Context, req *clientpb.VoteReviewRequest) (*clientpb.VoteReviewResponse, error) {
	review, err := h.service.VoteReview(ctx, req.GetReviewId(), req.GetUserId(), req.GetVote())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &clientpb.VoteReviewResponse{Review: mapper.ReviewToProto(review)}, nil
}

func (h *Handler) ReportReview(ctx context.Context, req *clientpb.ReportReviewRequest) (*clientpb.ReportReviewResponse, error) {
	report, err := h.service.ReportReview(ctx, req.GetReviewId(), req.GetReporterUserId(), req.GetReason(), req.GetDetails())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &clientpb.ReportReviewResponse{Report: mapper.ReportToProto(report)}, nil
}

func toStatusError(err error) error {
	switch {
	case errors.Is(err, reviewerrors.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, reviewerrors.ErrReviewNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, reviewerrors.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, reviewerrors.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, reviewerrors.ErrNoPurchase):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return err
	}
}
