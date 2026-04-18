package adminauth

import (
	"context"
	"time"
)

type Writer interface {
	UpdateLastLoginAt(ctx context.Context, id string, at time.Time) error
}

type writer struct {
	repository Repository
}

func NewWriter(repository Repository) Writer {
	return &writer{repository: repository}
}

func (w *writer) UpdateLastLoginAt(ctx context.Context, id string, at time.Time) error {
	return w.repository.UpdateLastLoginAt(ctx, id, at)
}
