package batch

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"

	"github.com/ljj/gugu-admin-api/internal/core/enum"
)

type JobType string

const (
	JobTypePriceUpdate JobType = "PRICE_UPDATE"
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

type PriceUpdateFilter struct {
	CollectionSource string      `json:"collection_source,omitempty"`
	Market           enum.Market `json:"market,omitempty"`
	ProductIDs       []string    `json:"product_ids,omitempty"`
	Currencies       []string    `json:"currencies,omitempty"`
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

type PriceHistoryRecorder interface {
	InsertProductPrice(ctx context.Context, productID string, recordedAt time.Time, price, currency, changeValue string) error
	GetLatestProductPrice(ctx context.Context, productID, currency string) (string, error)
	UpsertProductSnapshot(ctx context.Context, productID string, snapshotDate time.Time, price, currency string) error
}

type ProductVariantWriter interface {
	UpsertProductVariant(ctx context.Context, productID, language, currency, title, mainImageURL, productURL, currentPrice string, collectedAt time.Time) error
}

type PriceUpdater struct {
	productService *domainproduct.Service
	statusStore    *BatchStatusStore
	priceSource    ProductPriceSource
	priceRecorder  PriceHistoryRecorder
	variantWriter  ProductVariantWriter
	runMu          sync.Mutex
}

func NewPriceUpdater(
	productService *domainproduct.Service,
	statusStore *BatchStatusStore,
	priceSource ProductPriceSource,
	priceRecorder PriceHistoryRecorder,
	variantWriter ProductVariantWriter,
) *PriceUpdater {
	return &PriceUpdater{
		productService: productService,
		statusStore:    statusStore,
		priceSource:    priceSource,
		priceRecorder:  priceRecorder,
		variantWriter:  variantWriter,
	}
}

func (u *PriceUpdater) Preview(ctx context.Context, req PriceUpdateRequest) (*BatchJobStatus, error) {
	req = normalizePriceUpdateRequest(req)

	targets, err := u.resolveTargets(ctx, req.Filter)
	if err != nil {
		return nil, err
	}

	status := BatchJobStatus{
		JobType:     JobTypePriceUpdate,
		Status:      JobStatusQueued,
		TriggerType: req.TriggerType,
		Filter:      req.Filter,
		RequestedAt: req.RequestedAt,
		TotalCount:  len(targets),
	}
	u.statusStore.Set(status)

	return &status, nil
}

func (u *PriceUpdater) CurrentStatus() (BatchJobStatus, bool) {
	return u.statusStore.Get(JobTypePriceUpdate)
}

func (u *PriceUpdater) Run(ctx context.Context, req PriceUpdateRequest) (*PriceUpdateResult, error) {
	u.runMu.Lock()
	defer u.runMu.Unlock()

	if u.priceSource == nil {
		return nil, fmt.Errorf("price source is not configured")
	}

	req = normalizePriceUpdateRequest(req)

	targets, err := u.resolveTargets(ctx, req.Filter)
	if err != nil {
		return nil, err
	}

	startedAt := time.Now()
	runningStatus := BatchJobStatus{
		JobType:     JobTypePriceUpdate,
		Status:      JobStatusRunning,
		TriggerType: req.TriggerType,
		Filter:      req.Filter,
		RequestedAt: req.RequestedAt,
		StartedAt:   &startedAt,
		TotalCount:  len(targets),
	}
	u.statusStore.Set(runningStatus)

	result := &PriceUpdateResult{
		JobType:    JobTypePriceUpdate,
		Status:     JobStatusRunning,
		StartedAt:  startedAt,
		TotalCount: len(targets),
	}

	for i, product := range targets {
		if ctx.Err() != nil {
			result.Status = JobStatusFailed
			result.LastError = ctx.Err().Error()
			break
		}

		log.Printf("[price-update %d/%d] loading price for product=%s external=%s", i+1, len(targets), product.ID, product.ExternalProductID)

		anyUpdated := false
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		currencies := currenciesForProduct(product, req.Filter)
		masterCurrency := normalizeRepresentativeCurrency(product.Currency)

		for _, currency := range currencies {
			payload, err := u.priceSource.Load(ctx, product, currency)
			if err != nil {
				log.Printf("[price-update %d/%d] FAILED load currency=%s: %v", i+1, len(targets), currency, err)
				continue
			}

			if u.variantWriter != nil {
				if err := u.variantWriter.UpsertProductVariant(
					ctx,
					product.ID,
					enum.LanguageForCurrency(currency),
					currency,
					payload.Title,
					payload.MainImageURL,
					payload.ProductURL,
					payload.CurrentPrice,
					now,
				); err != nil {
					log.Printf("[price-update %d/%d] FAILED variant currency=%s: %v", i+1, len(targets), currency, err)
				}
			}

			// product 마스터는 가격을 저장하지 않고, 대표 메타와 수집 시각만 갱신한다.
			if currency == masterCurrency {
				_, changed, err := u.productService.RefreshCollectedMetadata(
					ctx,
					product.ID,
					payload.Title,
					payload.MainImageURL,
					payload.ProductURL,
					now,
				)
				if err != nil {
					log.Printf("[price-update %d/%d] FAILED update currency=%s: %v", i+1, len(targets), currency, err)
					continue
				}
				if changed {
					anyUpdated = true
				}
			}

			// price_history + snapshot 기록 (모든 통화)
			if u.priceRecorder != nil {
				changeValue := ""
				lastPrice, _ := u.priceRecorder.GetLatestProductPrice(ctx, product.ID, currency)
				shouldInsertHistory := lastPrice == ""
				if lastPrice != "" && lastPrice != payload.CurrentPrice {
					shouldInsertHistory = true
					changeValue = calcChange(lastPrice, payload.CurrentPrice)
				}

				if shouldInsertHistory {
					if err := u.priceRecorder.InsertProductPrice(ctx, product.ID, now, payload.CurrentPrice, currency, changeValue); err != nil {
						log.Printf("[price-update %d/%d] FAILED history currency=%s: %v", i+1, len(targets), currency, err)
					}
				}
				if err := u.priceRecorder.UpsertProductSnapshot(ctx, product.ID, today, payload.CurrentPrice, currency); err != nil {
					log.Printf("[price-update %d/%d] FAILED snapshot currency=%s: %v", i+1, len(targets), currency, err)
				}
			}

			log.Printf("[price-update %d/%d] OK currency=%s price=%s source=%s", i+1, len(targets), currency, payload.CurrentPrice, payload.PriceSource)
		}

		if anyUpdated {
			result.SuccessCount++
		} else {
			result.SkippedCount++
		}

		u.updateStatusFromResult(req, result, nil)
	}

	finishedAt := time.Now()
	result.FinishedAt = &finishedAt
	if result.Status != JobStatusFailed {
		result.Status = JobStatusCompleted
	}

	u.updateStatusFromResult(req, result, &finishedAt)
	return result, nil
}

func (u *PriceUpdater) resolveTargets(ctx context.Context, filter PriceUpdateFilter) ([]domainproduct.Product, error) {
	if len(filter.ProductIDs) > 0 {
		products, err := u.productService.FindByIDs(ctx, filter.ProductIDs)
		if err != nil {
			return nil, fmt.Errorf("find products by ids: %w", err)
		}
		return filterProducts(products, filter), nil
	}

	products, err := u.productService.ListPriceUpdateCandidates(ctx, domainproduct.PriceUpdateCandidateFilter{
		CollectionSource: filter.CollectionSource,
		Market:           filter.Market,
		CollectedBefore:  filter.CollectedBefore,
	})
	if err != nil {
		return nil, fmt.Errorf("list price update candidates: %w", err)
	}
	return filterProducts(products, filter), nil
}

func (u *PriceUpdater) updateStatusFromResult(req PriceUpdateRequest, result *PriceUpdateResult, finishedAt *time.Time) {
	status := BatchJobStatus{
		JobType:      result.JobType,
		Status:       result.Status,
		TriggerType:  req.TriggerType,
		Filter:       req.Filter,
		RequestedAt:  req.RequestedAt,
		StartedAt:    &result.StartedAt,
		FinishedAt:   finishedAt,
		TotalCount:   result.TotalCount,
		SuccessCount: result.SuccessCount,
		FailCount:    result.FailCount,
		SkippedCount: result.SkippedCount,
		LastError:    result.LastError,
	}
	u.statusStore.Set(status)
}

func normalizePriceUpdateRequest(req PriceUpdateRequest) PriceUpdateRequest {
	if req.JobType == "" {
		req.JobType = JobTypePriceUpdate
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
	return req
}

func normalizeRepresentativeCurrency(currency string) string {
	currency = strings.TrimSpace(currency)
	if currency == "" {
		return enum.SupportedCurrencies[0]
	}
	if slices.Contains(enum.SupportedCurrencies, currency) {
		return currency
	}
	return enum.SupportedCurrencies[0]
}

func currenciesForProduct(product domainproduct.Product, filter PriceUpdateFilter) []string {
	if len(filter.Currencies) > 0 {
		return filter.Currencies
	}
	if product.CollectionSource == domainproduct.CollectionSourceHotProductQuery {
		return enum.SupportedCurrencies
	}
	return []string{normalizeRepresentativeCurrency(product.Currency)}
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
		if !slices.Contains(enum.SupportedCurrencies, trimmed) {
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

func calcChange(oldPrice, newPrice string) string {
	oldVal, err1 := strconv.ParseFloat(oldPrice, 64)
	newVal, err2 := strconv.ParseFloat(newPrice, 64)
	if err1 != nil || err2 != nil {
		return ""
	}
	diff := newVal - oldVal
	return strconv.FormatFloat(diff, 'f', -1, 64)
}

func filterProducts(products []domainproduct.Product, filter PriceUpdateFilter) []domainproduct.Product {
	filtered := make([]domainproduct.Product, 0, len(products))
	for _, product := range products {
		if filter.CollectionSource != "" && product.CollectionSource != filter.CollectionSource {
			continue
		}
		if filter.Market != "" && product.Market != filter.Market {
			continue
		}
		if len(filter.ProductIDs) > 0 && !slices.Contains(filter.ProductIDs, product.ID) {
			continue
		}
		filtered = append(filtered, product)
	}
	return filtered
}
