package product

import (
	"time"

	"github.com/ljj/gugu-admin-api/internal/core/enum"
)

type PriceUpdateCandidateFilter struct {
	CollectionSource string
	Market           enum.Market
	CollectedBefore  *time.Time
}
