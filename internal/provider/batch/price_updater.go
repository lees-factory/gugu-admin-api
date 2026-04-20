package batch

import (
	"strconv"
	"strings"
	"sync"
	"time"

	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"

	"github.com/ljj/gugu-admin-api/internal/core/enum"
)

type JobType string

const (
	JobTypeSKUSnapshotUpdate   JobType = "SKU_SNAPSHOT_UPDATE"
	JobTypePriceAlertEmailScan JobType = "PRICE_ALERT_EMAIL_SCAN"
)

type TriggerType string

const (
	TriggerTypeManual    TriggerType = "MANUAL"
	TriggerTypeScheduled TriggerType = "SCHEDULED"
)

type JobStatus string

const (
	JobStatusQueued    JobStatus = "QUEUED"
	JobStatusRunning   JobStatus = "RUNNING"
	JobStatusCompleted JobStatus = "COMPLETED"
	JobStatusFailed    JobStatus = "FAILED"
)

type TargetGroup string

const (
	TargetGroupAll         TargetGroup = "ALL"
	TargetGroupHotProducts TargetGroup = "HOT_PRODUCTS"
	TargetGroupTracked     TargetGroup = "TRACKED"
)

type PriceUpdateFilter struct {
	CollectionSource string      `json:"collection_source,omitempty"`
	Market           enum.Market `json:"market,omitempty"`
	ProductIDs       []string    `json:"product_ids,omitempty"`
	Currencies       []string    `json:"currencies,omitempty"`
	TargetGroup      TargetGroup `json:"target_group,omitempty"`
	CollectedBefore  *time.Time  `json:"collected_before,omitempty"`
	Force            bool        `json:"force"`
}

type PriceUpdateRequest struct {
	JobType     JobType           `json:"job_type"`
	TriggerType TriggerType       `json:"trigger_type"`
	Filter      PriceUpdateFilter `json:"filter"`
	RequestedAt time.Time         `json:"requested_at"`
	RequestedBy string            `json:"requested_by,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type PriceUpdateResult struct {
	JobType      JobType    `json:"job_type"`
	Status       JobStatus  `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	TotalCount   int        `json:"total_count"`
	SuccessCount int        `json:"success_count"`
	FailCount    int        `json:"fail_count"`
	SkippedCount int        `json:"skipped_count"`
	LastError    string     `json:"last_error,omitempty"`
}

type BatchJobStatus struct {
	JobType      JobType           `json:"job_type"`
	Status       JobStatus         `json:"status"`
	TriggerType  TriggerType       `json:"trigger_type"`
	Filter       PriceUpdateFilter `json:"filter"`
	RequestedAt  time.Time         `json:"requested_at"`
	StartedAt    *time.Time        `json:"started_at,omitempty"`
	FinishedAt   *time.Time        `json:"finished_at,omitempty"`
	TotalCount   int               `json:"total_count"`
	SuccessCount int               `json:"success_count"`
	FailCount    int               `json:"fail_count"`
	SkippedCount int               `json:"skipped_count"`
	LastError    string            `json:"last_error,omitempty"`
}

type BatchStatusStore struct {
	mu     sync.RWMutex
	latest map[JobType]BatchJobStatus
}

func NewBatchStatusStore() *BatchStatusStore {
	return &BatchStatusStore{
		latest: make(map[JobType]BatchJobStatus),
	}
}

func (s *BatchStatusStore) Set(status BatchJobStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest[status.JobType] = status
}

func (s *BatchStatusStore) Get(jobType JobType) (BatchJobStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.latest[jobType]
	return status, ok
}

func normalizePriceUpdateRequest(req PriceUpdateRequest) PriceUpdateRequest {
	if req.JobType == "" {
		req.JobType = JobTypeSKUSnapshotUpdate
	}
	if req.TriggerType == "" {
		req.TriggerType = TriggerTypeManual
	}
	if req.RequestedAt.IsZero() {
		req.RequestedAt = time.Now()
	}
	req.Filter.CollectionSource = strings.TrimSpace(req.Filter.CollectionSource)
	req.Filter.ProductIDs = compactStrings(req.Filter.ProductIDs)
	req.Filter.Currencies = normalizeRequestedCurrencies(req.Filter.Currencies)
	req.Filter.TargetGroup = normalizeTargetGroup(req.Filter.TargetGroup)
	return req
}

func normalizeRepresentativeCurrency(currency string) string {
	currency = strings.TrimSpace(currency)
	if currency == "" {
		return enum.SupportedCurrencies[0]
	}
	for _, supported := range enum.SupportedCurrencies {
		if supported == currency {
			return currency
		}
	}
	return enum.SupportedCurrencies[0]
}

func currenciesForProduct(_ domainproduct.Product, filter PriceUpdateFilter) []string {
	if len(filter.Currencies) > 0 {
		return filter.Currencies
	}
	return []string{enum.SupportedCurrencies[0]}
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func normalizeRequestedCurrencies(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		trimmed := strings.ToUpper(strings.TrimSpace(value))
		if trimmed == "" {
			continue
		}
		allowed := false
		for _, supported := range enum.SupportedCurrencies {
			if supported == trimmed {
				allowed = true
				break
			}
		}
		if !allowed {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}

func normalizeTargetGroup(group TargetGroup) TargetGroup {
	switch TargetGroup(strings.ToUpper(strings.TrimSpace(string(group)))) {
	case TargetGroupHotProducts:
		return TargetGroupHotProducts
	case TargetGroupTracked:
		return TargetGroupTracked
	default:
		return TargetGroupAll
	}
}

func filterProducts(products []domainproduct.Product, filter PriceUpdateFilter) []domainproduct.Product {
	if len(products) == 0 {
		return products
	}

	result := make([]domainproduct.Product, 0, len(products))
	for _, p := range products {
		if filter.CollectionSource != "" && p.CollectionSource != filter.CollectionSource {
			continue
		}
		if filter.Market != "" && p.Market != filter.Market {
			continue
		}
		result = append(result, p)
	}
	return result
}

func calcChange(oldPrice, newPrice string) string {
	oldVal, err1 := strconv.ParseFloat(oldPrice, 64)
	newVal, err2 := strconv.ParseFloat(newPrice, 64)
	if err1 != nil || err2 != nil {
		return ""
	}
	diff := newVal - oldVal
	return strconv.FormatFloat(diff, 'f', -1, 64)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
