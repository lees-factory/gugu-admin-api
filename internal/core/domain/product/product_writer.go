package product

import "context"

type Writer interface {
	Create(ctx context.Context, product Product) error
	Update(ctx context.Context, product Product) error
}

type writer struct {
	repository Repository
}

func NewWriter(repository Repository) Writer {
	return &writer{repository: repository}
}

func (w *writer) Create(ctx context.Context, product Product) error {
	return w.repository.Create(ctx, product)
}

func (w *writer) Update(ctx context.Context, product Product) error {
	return w.repository.Update(ctx, product)
}
