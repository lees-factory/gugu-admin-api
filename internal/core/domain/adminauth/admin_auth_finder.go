package adminauth

import "context"

type Finder interface {
	GetByID(ctx context.Context, id string) (*AdminUser, error)
	GetByLoginID(ctx context.Context, loginID string) (*AdminUser, error)
}

type finder struct {
	repository Repository
}

func NewFinder(repository Repository) Finder {
	return &finder{repository: repository}
}

func (f *finder) GetByID(ctx context.Context, id string) (*AdminUser, error) {
	return f.repository.GetByID(ctx, id)
}

func (f *finder) GetByLoginID(ctx context.Context, loginID string) (*AdminUser, error) {
	return f.repository.GetByLoginID(ctx, loginID)
}
