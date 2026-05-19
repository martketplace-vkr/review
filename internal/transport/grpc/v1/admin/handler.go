package admin

import (
	"context"
	"errors"

	"github.com/martketplace-vkr/review/internal/service/reviewerrors"
	"github.com/martketplace-vkr/review/internal/transport/grpc/v1/mapper"
	adminpb "github.com/martketplace-vkr/review/pkg/api/grpc/v1/admin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	service service
	adminpb.UnimplementedReviewAdminServiceServer
}

func New(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListReviewDisputes(ctx context.Context, req *adminpb.ListReviewDisputesRequest) (*adminpb.ListReviewDisputesResponse, error) {
	reviews, err := h.service.ListReviewDisputes(ctx, req.GetStatus())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &adminpb.ListReviewDisputesResponse{Reviews: mapper.ReviewsToProto(reviews)}, nil
}

func (h *Handler) ListReviewReports(ctx context.Context, req *adminpb.ListReviewReportsRequest) (*adminpb.ListReviewReportsResponse, error) {
	reports, err := h.service.ListReviewReports(ctx, req.GetStatus())
	if err != nil {
		return nil, toStatusError(err)
	}

	items := make([]*adminpb.ReportedReview, 0, len(reports))
	for _, report := range reports {
		items = append(items, &adminpb.ReportedReview{
			Review: mapper.ReviewToProto(&report.Review),
			Report: mapper.ReportToProto(&report.Report),
		})
	}

	return &adminpb.ListReviewReportsResponse{Items: items}, nil
}

func (h *Handler) ResolveReviewDispute(ctx context.Context, req *adminpb.ResolveReviewDisputeRequest) (*adminpb.ResolveReviewDisputeResponse, error) {
	review, err := h.service.ResolveReviewDispute(ctx, req.GetAdminId(), req.GetReviewId(), req.GetDisputeId(), req.GetDecision(), req.GetComment())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &adminpb.ResolveReviewDisputeResponse{Review: mapper.ReviewToProto(review)}, nil
}

func (h *Handler) DeleteReview(ctx context.Context, req *adminpb.DeleteReviewRequest) (*adminpb.DeleteReviewResponse, error) {
	if err := h.service.DeleteReview(ctx, req.GetAdminId(), req.GetReviewId()); err != nil {
		return nil, toStatusError(err)
	}

	return &adminpb.DeleteReviewResponse{Deleted: true}, nil
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
