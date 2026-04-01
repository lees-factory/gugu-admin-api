package user

import "context"

type Repository interface {
	List(ctx context.Context, filter ListFilter) ([]User, error)
	Count(ctx context.Context, filter ListFilter) (int64, error)
}
