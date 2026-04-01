package user

import (
	"context"
	"strings"
)

type Service struct {
	finder Finder
}

func NewService(finder Finder) *Service {
	return &Service{finder: finder}
}

func (s *Service) List(ctx context.Context, filter ListFilter) (*ListResult, error) {
	filter.Search = strings.TrimSpace(filter.Search)
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	totalCount, err := s.finder.Count(ctx, filter)
	if err != nil {
		return nil, err
	}

	users, err := s.finder.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &ListResult{
		TotalCount: totalCount,
		Users:      users,
	}, nil
}
