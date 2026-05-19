package mapper

import (
	"time"

	"github.com/martketplace-vkr/review/domain"
	domainpb "github.com/martketplace-vkr/review/pkg/api/grpc/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ReviewToProto(review *domain.Review) *domainpb.Review {
	if review == nil {
		return nil
	}

	return &domainpb.Review{
		Id:                 review.ID,
		ProductId:          review.ProductID,
		VendorId:           review.VendorID,
		AuthorUserId:       review.AuthorUserID,
		AuthorName:         review.AuthorName,
		Rating:             review.Rating,
		Comment:            review.Comment,
		ExcludedFromRating: review.ExcludedFromRating,
		HelpfulCount:       review.HelpfulCount,
		NotHelpfulCount:    review.NotHelpfulCount,
		Images:             ImagesToProto(review.Images),
		Reply:              ReplyToProto(review.Reply),
		Dispute:            DisputeToProto(review.Dispute),
		CreatedAt:          newTimestamp(review.CreatedAt),
		UpdatedAt:          newTimestamp(review.UpdatedAt),
	}
}

func ReviewsToProto(reviews []domain.Review) []*domainpb.Review {
	result := make([]*domainpb.Review, 0, len(reviews))
	for index := range reviews {
		result = append(result, ReviewToProto(&reviews[index]))
	}

	return result
}

func ImagesToProto(images []domain.Image) []*domainpb.ReviewImage {
	result := make([]*domainpb.ReviewImage, 0, len(images))
	for _, image := range images {
		result = append(result, &domainpb.ReviewImage{
			Id:        image.ID,
			Url:       image.URL,
			SortOrder: image.SortOrder,
		})
	}

	return result
}

func ReplyToProto(reply *domain.Reply) *domainpb.ReviewReply {
	if reply == nil {
		return nil
	}

	return &domainpb.ReviewReply{
		Id:        reply.ID,
		VendorId:  reply.VendorID,
		Comment:   reply.Comment,
		CreatedAt: newTimestamp(reply.CreatedAt),
		UpdatedAt: newTimestamp(reply.UpdatedAt),
	}
}

func DisputeToProto(dispute *domain.Dispute) *domainpb.ReviewDispute {
	if dispute == nil {
		return nil
	}

	return &domainpb.ReviewDispute{
		Id:           dispute.ID,
		VendorId:     dispute.VendorID,
		Reason:       dispute.Reason,
		Status:       dispute.Status,
		AdminComment: dispute.AdminComment,
		CreatedAt:    newTimestamp(dispute.CreatedAt),
		ResolvedAt:   newOptionalTimestamp(dispute.ResolvedAt),
	}
}

func RatingSummaryToProto(summary *domain.RatingSummary) *domainpb.RatingSummary {
	if summary == nil {
		return nil
	}

	return &domainpb.RatingSummary{
		ProductId:      summary.ProductID,
		AverageRating:  summary.AverageRating,
		RatingCount:    summary.RatingCount,
		OneStarCount:   summary.OneStarCount,
		TwoStarCount:   summary.TwoStarCount,
		ThreeStarCount: summary.ThreeStarCount,
		FourStarCount:  summary.FourStarCount,
		FiveStarCount:  summary.FiveStarCount,
	}
}

func ReportToProto(report *domain.Report) *domainpb.ReviewReport {
	if report == nil {
		return nil
	}

	return &domainpb.ReviewReport{
		Id:             report.ID,
		ReviewId:       report.ReviewID,
		ReporterUserId: report.ReporterUserID,
		Reason:         report.Reason,
		Details:        report.Details,
		Status:         report.Status,
		CreatedAt:      newTimestamp(report.CreatedAt),
	}
}

func newTimestamp(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}

	return timestamppb.New(value)
}

func newOptionalTimestamp(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}

	return newTimestamp(*value)
}
