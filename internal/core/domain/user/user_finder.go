package user

import "context"

type Finder interface {
	List(ctx context.Context, filter ListFilter) ([]User, error)
	Count(ctx context.Context, filter ListFilter) (int64, error)
}

type finder struct {
	repository Repository
}

func NewFinder(repository Repository) Finder {
	return &finder{repository: repository}
}

func (f *finder) List(ctx context.Context, filter ListFilter) ([]User, error) {
	return f.repository.List(ctx, filter)
}

func (f *finder) Count(ctx context.Context, filter ListFilter) (int64, error) {
	return f.repository.Count(ctx, filter)
}
