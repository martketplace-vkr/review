package domain

import "time"

const (
	VoteHelpful    = "helpful"
	VoteNotHelpful = "not_helpful"

	ReportContent = "content"
	ReportMedia   = "media"
	ReportSpam    = "spam"

	DisputePending  = "pending"
	DisputeAccepted = "accepted"
	DisputeRejected = "rejected"
)

type Review struct {
	ID                 int64      `db:"id"`
	ProductID          int64      `db:"product_id"`
	VendorID           int64      `db:"vendor_id"`
	AuthorUserID       int64      `db:"author_user_id"`
	AuthorName         string     `db:"author_name"`
	Rating             int32      `db:"rating"`
	Comment            string     `db:"comment"`
	ExcludedFromRating bool       `db:"excluded_from_rating"`
	HelpfulCount       int64      `db:"helpful_count"`
	NotHelpfulCount    int64      `db:"not_helpful_count"`
	Images             []Image    `db:"-"`
	Reply              *Reply     `db:"-"`
	Dispute            *Dispute   `db:"-"`
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
	DeletedAt          *time.Time `db:"deleted_at"`
}

type Image struct {
	ID        int64  `db:"id"`
	ReviewID  int64  `db:"review_id"`
	URL       string `db:"url"`
	SortOrder uint32 `db:"sort_order"`
}

type Reply struct {
	ID        int64     `db:"id"`
	ReviewID  int64     `db:"review_id"`
	VendorID  int64     `db:"vendor_id"`
	Comment   string    `db:"comment"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Dispute struct {
	ID           int64      `db:"id"`
	ReviewID     int64      `db:"review_id"`
	VendorID     int64      `db:"vendor_id"`
	Reason       string     `db:"reason"`
	Status       string     `db:"status"`
	AdminComment string     `db:"admin_comment"`
	CreatedAt    time.Time  `db:"created_at"`
	ResolvedAt   *time.Time `db:"resolved_at"`
}

type Report struct {
	ID             int64     `db:"id"`
	ReviewID       int64     `db:"review_id"`
	ReporterUserID int64     `db:"reporter_user_id"`
	Reason         string    `db:"reason"`
	Details        string    `db:"details"`
	Status         string    `db:"status"`
	CreatedAt      time.Time `db:"created_at"`
}

type ReportedReview struct {
	Review Review
	Report Report
}

type RatingSummary struct {
	ProductID      int64   `db:"product_id"`
	AverageRating  float64 `db:"average_rating"`
	RatingCount    int64   `db:"rating_count"`
	OneStarCount   int64   `db:"one_star_count"`
	TwoStarCount   int64   `db:"two_star_count"`
	ThreeStarCount int64   `db:"three_star_count"`
	FourStarCount  int64   `db:"four_star_count"`
	FiveStarCount  int64   `db:"five_star_count"`
}
