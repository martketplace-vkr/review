package vendor

import (
	"context"
	"errors"

	"github.com/martketplace-vkr/review/internal/service/reviewerrors"
	"github.com/martketplace-vkr/review/internal/transport/grpc/v1/mapper"
	vendorpb "github.com/martketplace-vkr/review/pkg/api/grpc/v1/vendor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	service service
	vendorpb.UnimplementedReviewVendorServiceServer
}

func New(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListVendorReviews(ctx context.Context, req *vendorpb.ListVendorReviewsRequest) (*vendorpb.ListVendorReviewsResponse, error) {
	reviews, err := h.service.ListVendorReviews(ctx, req.GetVendorId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &vendorpb.ListVendorReviewsResponse{Reviews: mapper.ReviewsToProto(reviews)}, nil
}

func (h *Handler) ReplyReview(ctx context.Context, req *vendorpb.ReplyReviewRequest) (*vendorpb.ReplyReviewResponse, error) {
	review, err := h.service.ReplyReview(ctx, req.GetVendorId(), req.GetReviewId(), req.GetComment())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &vendorpb.ReplyReviewResponse{Review: mapper.ReviewToProto(review)}, nil
}

func (h *Handler) DisputeReview(ctx context.Context, req *vendorpb.DisputeReviewRequest) (*vendorpb.DisputeReviewResponse, error) {
	review, err := h.service.DisputeReview(ctx, req.GetVendorId(), req.GetReviewId(), req.GetReason())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &vendorpb.DisputeReviewResponse{Review: mapper.ReviewToProto(review)}, nil
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
