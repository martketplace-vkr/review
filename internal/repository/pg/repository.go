package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/martketplace-vkr/review/domain"
)

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateReview(ctx context.Context, review domain.Review, imageURLs []string) (*domain.Review, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		insert into review.reviews (
			product_id,
			vendor_id,
			author_user_id,
			author_name,
			rating,
			comment
		) values ($1, $2, $3, $4, $5, $6)
		returning
			id,
			product_id,
			vendor_id,
			author_user_id,
			author_name,
			rating,
			comment,
			excluded_from_rating,
			0::bigint as helpful_count,
			0::bigint as not_helpful_count,
			created_at,
			updated_at,
			deleted_at
	`

	created := new(domain.Review)
	err = tx.GetContext(ctx, created, query,
		review.ProductID,
		review.VendorID,
		review.AuthorUserID,
		review.AuthorName,
		review.Rating,
		review.Comment,
	)
	if err != nil {
		return nil, err
	}

	imageQuery := `
		insert into review.review_images (review_id, url, sort_order)
		values ($1, $2, $3)
	`
	for index, imageURL := range imageURLs {
		if strings.TrimSpace(imageURL) == "" {
			continue
		}

		if _, err = tx.ExecContext(ctx, imageQuery, created.ID, strings.TrimSpace(imageURL), index); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetReview(ctx, created.ID)
}

func (r *Repository) ListProductReviews(ctx context.Context, productID int64) ([]domain.Review, error) {
	return r.listReviews(ctx, `
		where r.product_id = $1
			and r.deleted_at is null
		order by (coalesce(v.helpful_count, 0) - coalesce(v.not_helpful_count, 0)) desc, r.created_at desc, r.id desc
	`, productID)
}

func (r *Repository) ListVendorReviews(ctx context.Context, vendorID int64) ([]domain.Review, error) {
	return r.listReviews(ctx, `
		where r.vendor_id = $1
			and r.deleted_at is null
		order by r.created_at desc, r.id desc
	`, vendorID)
}

func (r *Repository) ListDisputedReviews(ctx context.Context, status string) ([]domain.Review, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		status = domain.DisputePending
	}

	return r.listReviews(ctx, `
		join review.review_disputes filter_dispute on filter_dispute.review_id = r.id
			and filter_dispute.status = $1
		where r.deleted_at is null
		order by filter_dispute.created_at asc, r.created_at desc
	`, status)
}

func (r *Repository) GetReview(ctx context.Context, reviewID int64) (*domain.Review, error) {
	reviews, err := r.listReviews(ctx, `
		where r.id = $1
			and r.deleted_at is null
	`, reviewID)
	if err != nil {
		return nil, err
	}
	if len(reviews) == 0 {
		return nil, sql.ErrNoRows
	}

	return &reviews[0], nil
}

func (r *Repository) UpsertVote(ctx context.Context, reviewID int64, userID int64, vote string) (*domain.Review, error) {
	query := `
		insert into review.review_votes (review_id, user_id, vote)
		values ($1, $2, $3)
		on conflict (review_id, user_id)
		do update set
			vote = excluded.vote,
			updated_at = now()
	`
	if _, err := r.db.ExecContext(ctx, query, reviewID, userID, vote); err != nil {
		return nil, err
	}

	return r.GetReview(ctx, reviewID)
}

func (r *Repository) CreateReport(ctx context.Context, report domain.Report) (*domain.Report, error) {
	query := `
		insert into review.review_reports (
			review_id,
			reporter_user_id,
			reason,
			details
		) values ($1, $2, $3, $4)
		on conflict (review_id, reporter_user_id)
		do update set
			reason = excluded.reason,
			details = excluded.details,
			status = 'pending',
			updated_at = now()
		returning
			id,
			review_id,
			reporter_user_id,
			reason,
			details,
			status,
			created_at
	`

	created := new(domain.Report)
	err := r.db.GetContext(ctx, created, query,
		report.ReviewID,
		report.ReporterUserID,
		report.Reason,
		report.Details,
	)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *Repository) UpsertReply(ctx context.Context, reviewID int64, vendorID int64, comment string) (*domain.Review, error) {
	query := `
		insert into review.review_replies (review_id, vendor_id, comment)
		select id, vendor_id, $3
		from review.reviews
		where id = $1
			and vendor_id = $2
			and deleted_at is null
		on conflict (review_id)
		do update set
			comment = excluded.comment,
			deleted_at = null,
			updated_at = now()
	`

	result, err := r.db.ExecContext(ctx, query, reviewID, vendorID, comment)
	if err != nil {
		return nil, err
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, sql.ErrNoRows
	}

	return r.GetReview(ctx, reviewID)
}

func (r *Repository) CreateDispute(ctx context.Context, reviewID int64, vendorID int64, reason string) (*domain.Review, error) {
	query := `
		insert into review.review_disputes (review_id, vendor_id, reason)
		select id, vendor_id, $3
		from review.reviews
		where id = $1
			and vendor_id = $2
			and deleted_at is null
		on conflict (review_id, vendor_id) where status = 'pending'
		do update set
			reason = excluded.reason
	`

	result, err := r.db.ExecContext(ctx, query, reviewID, vendorID, reason)
	if err != nil {
		return nil, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, sql.ErrNoRows
	}

	return r.GetReview(ctx, reviewID)
}

func (r *Repository) ResolveDispute(ctx context.Context, reviewID int64, disputeID int64, adminID int64, accepted bool, comment string) (*domain.Review, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	status := domain.DisputeRejected
	if accepted {
		status = domain.DisputeAccepted
	}

	result, err := tx.ExecContext(ctx, `
		update review.review_disputes
		set
			status = $4,
			admin_id = $5,
			admin_comment = $6,
			resolved_at = now()
		where id = $1
			and review_id = $2
			and status = $3
	`, disputeID, reviewID, domain.DisputePending, status, adminID, comment)
	if err != nil {
		return nil, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, sql.ErrNoRows
	}

	if accepted {
		if _, err = tx.ExecContext(ctx, `
			update review.reviews
			set
				excluded_from_rating = true,
				updated_at = now()
			where id = $1
				and deleted_at is null
		`, reviewID); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetReview(ctx, reviewID)
}

func (r *Repository) DeleteReview(ctx context.Context, reviewID int64, adminID int64) error {
	result, err := r.db.ExecContext(ctx, `
		update review.reviews
		set
			deleted_at = now(),
			deleted_by_admin_id = $2,
			updated_at = now()
		where id = $1
			and deleted_at is null
	`, reviewID, adminID)
	if err != nil {
		return err
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *Repository) RatingSummary(ctx context.Context, productID int64) (*domain.RatingSummary, error) {
	query := `
		select
			$1::bigint as product_id,
			coalesce(avg(rating), 0)::float8 as average_rating,
			count(*)::bigint as rating_count,
			count(*) filter (where rating = 1)::bigint as one_star_count,
			count(*) filter (where rating = 2)::bigint as two_star_count,
			count(*) filter (where rating = 3)::bigint as three_star_count,
			count(*) filter (where rating = 4)::bigint as four_star_count,
			count(*) filter (where rating = 5)::bigint as five_star_count
		from review.reviews
		where product_id = $1
			and deleted_at is null
			and excluded_from_rating = false
	`

	summary := new(domain.RatingSummary)
	if err := r.db.GetContext(ctx, summary, query, productID); err != nil {
		return nil, err
	}

	return summary, nil
}

func (r *Repository) listReviews(ctx context.Context, clause string, args ...any) ([]domain.Review, error) {
	query := fmt.Sprintf(`
		select
			r.id,
			r.product_id,
			r.vendor_id,
			r.author_user_id,
			r.author_name,
			r.rating,
			r.comment,
			r.excluded_from_rating,
			coalesce(v.helpful_count, 0)::bigint as helpful_count,
			coalesce(v.not_helpful_count, 0)::bigint as not_helpful_count,
			r.created_at,
			r.updated_at,
			r.deleted_at
		from review.reviews r
		left join (
			select
				review_id,
				count(*) filter (where vote = 'helpful') as helpful_count,
				count(*) filter (where vote = 'not_helpful') as not_helpful_count
			from review.review_votes
			group by review_id
		) v on v.review_id = r.id
		%s
	`, clause)

	reviews := make([]domain.Review, 0)
	if err := r.db.SelectContext(ctx, &reviews, query, args...); err != nil {
		return nil, err
	}

	if len(reviews) == 0 {
		return reviews, nil
	}

	if err := r.attachImages(ctx, reviews); err != nil {
		return nil, err
	}
	if err := r.attachReplies(ctx, reviews); err != nil {
		return nil, err
	}
	if err := r.attachLatestDisputes(ctx, reviews); err != nil {
		return nil, err
	}

	return reviews, nil
}

func (r *Repository) attachImages(ctx context.Context, reviews []domain.Review) error {
	ids := reviewIDs(reviews)
	query, args, err := sqlx.In(`
		select id, review_id, url, sort_order
		from review.review_images
		where review_id in (?)
		order by review_id asc, sort_order asc, id asc
	`, ids)
	if err != nil {
		return err
	}
	query = r.db.Rebind(query)

	images := make([]domain.Image, 0)
	if err = r.db.SelectContext(ctx, &images, query, args...); err != nil {
		return err
	}

	byReview := make(map[int64][]domain.Image)
	for _, image := range images {
		byReview[image.ReviewID] = append(byReview[image.ReviewID], image)
	}
	for index := range reviews {
		reviews[index].Images = byReview[reviews[index].ID]
	}

	return nil
}

func (r *Repository) attachReplies(ctx context.Context, reviews []domain.Review) error {
	ids := reviewIDs(reviews)
	query, args, err := sqlx.In(`
		select id, review_id, vendor_id, comment, created_at, updated_at
		from review.review_replies
		where review_id in (?)
			and deleted_at is null
	`, ids)
	if err != nil {
		return err
	}
	query = r.db.Rebind(query)

	replies := make([]domain.Reply, 0)
	if err = r.db.SelectContext(ctx, &replies, query, args...); err != nil {
		return err
	}

	byReview := make(map[int64]domain.Reply)
	for _, reply := range replies {
		byReview[reply.ReviewID] = reply
	}
	for index := range reviews {
		if reply, ok := byReview[reviews[index].ID]; ok {
			reviews[index].Reply = &reply
		}
	}

	return nil
}

func (r *Repository) attachLatestDisputes(ctx context.Context, reviews []domain.Review) error {
	ids := reviewIDs(reviews)
	query, args, err := sqlx.In(`
		select distinct on (review_id)
			id,
			review_id,
			vendor_id,
			reason,
			status,
			admin_comment,
			created_at,
			resolved_at
		from review.review_disputes
		where review_id in (?)
		order by review_id, created_at desc, id desc
	`, ids)
	if err != nil {
		return err
	}
	query = r.db.Rebind(query)

	disputes := make([]domain.Dispute, 0)
	if err = r.db.SelectContext(ctx, &disputes, query, args...); err != nil {
		return err
	}

	byReview := make(map[int64]domain.Dispute)
	for _, dispute := range disputes {
		byReview[dispute.ReviewID] = dispute
	}
	for index := range reviews {
		if dispute, ok := byReview[reviews[index].ID]; ok {
			reviews[index].Dispute = &dispute
		}
	}

	return nil
}

func reviewIDs(reviews []domain.Review) []int64 {
	ids := make([]int64, 0, len(reviews))
	for _, review := range reviews {
		ids = append(ids, review.ID)
	}

	return ids
}

func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	var interfaceErr interface{ SQLState() string }
	if errors.As(err, &interfaceErr) {
		return interfaceErr.SQLState() == "23505"
	}

	return strings.Contains(err.Error(), "duplicate key")
}
