package reviewerrors

import "errors"

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrReviewNotFound  = errors.New("review not found")
	ErrForbidden       = errors.New("forbidden")
	ErrAlreadyExists   = errors.New("review already exists")
	ErrNoPurchase      = errors.New("successful product order is required to create review")
)
