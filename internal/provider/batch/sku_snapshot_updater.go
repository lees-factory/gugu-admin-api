package batch

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ljj/gugu-admin-api/internal/clients/aliexpress"
	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
	"github.com/ljj/gugu-admin-api/internal/core/enum"
)

const (
	JobTypeSKUSnapshotUpdate JobType = "SKU_SNAPSHOT_UPDATE"
)

type SKUSnapshotUpdater struct {
	productService   *domainproduct.Service
	statusStore      *BatchStatusStore
	aliexpressClient aliexpress.Client
	skuRecorder      SKUPriceRecorder
	runMu            sync.Mutex
}

func NewSKUSnapshotUpdater(
	productService *domainproduct.Service,
	statusStore *BatchStatusStore,
	aliexpressClient aliexpress.Client,
	skuRecorder SKUPriceRecorder,
) *SKUSnapshotUpdater {
	return &SKUSnapshotUpdater{
		productService:   productService,
		statusStore:      statusStore,
		aliexpressClient: aliexpressClient,
		skuRecorder:      skuRecorder,
	}
}

func (u *SKUSnapshotUpdater) Preview(ctx context.Context, req PriceUpdateRequest) (*BatchJobStatus, error) {
	req = normalizeSKUSnapshotUpdateRequest(req)

	targets, err := u.resolveTargets(ctx, req.Filter)
	if err != nil {
		return nil, err
	}

	status := BatchJobStatus{
		JobType:     JobTypeSKUSnapshotUpdate,
		Status:      JobStatusQueued,
		TriggerType: req.TriggerType,
		Filter:      req.Filter,
		RequestedAt: req.RequestedAt,
		TotalCount:  len(targets),
	}
	u.statusStore.Set(status)

	return &status, nil
}

func (u *SKUSnapshotUpdater) CurrentStatus() (BatchJobStatus, bool) {
	return u.statusStore.Get(JobTypeSKUSnapshotUpdate)
}

func (u *SKUSnapshotUpdater) Run(ctx context.Context, req PriceUpdateRequest) (*PriceUpdateResult, error) {
	u.runMu.Lock()
	defer u.runMu.Unlock()

	if u.aliexpressClient == nil {
		return nil, fmt.Errorf("aliexpress client is not configured")
	}
	if u.skuRecorder == nil {
		return nil, fmt.Errorf("sku recorder is not configured")
	}

	req = normalizeSKUSnapshotUpdateRequest(req)

	targets, err := u.resolveTargets(ctx, req.Filter)
	if err != nil {
		return nil, err
	}

	startedAt := time.Now()
	runningStatus := BatchJobStatus{
		JobType:     JobTypeSKUSnapshotUpdate,
		Status:      JobStatusRunning,
		TriggerType: req.TriggerType,
		Filter:      req.Filter,
		RequestedAt: req.RequestedAt,
		StartedAt:   &startedAt,
		TotalCount:  len(targets),
	}
	u.statusStore.Set(runningStatus)

	result := &PriceUpdateResult{
		JobType:    JobTypeSKUSnapshotUpdate,
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

		log.Printf("[sku-snapshot-update %d/%d] loading skus for product=%s external=%s", i+1, len(targets), product.ID, product.ExternalProductID)

		updatedCount := 0
		for _, currency := range currenciesForProduct(product) {
			detail, err := u.loadDropshippingDetail(ctx, product, currency)
			if err != nil {
				log.Printf("[sku-snapshot-update %d/%d] FAILED load currency=%s: %v", i+1, len(targets), currency, err)
				continue
			}

			count, err := u.recordSKUPrices(ctx, product.ID, detail, currency)
			if err != nil {
				log.Printf("[sku-snapshot-update %d/%d] FAILED record currency=%s: %v", i+1, len(targets), currency, err)
				continue
			}
			updatedCount += count
			log.Printf("[sku-snapshot-update %d/%d] OK currency=%s sku_count=%d", i+1, len(targets), currency, count)
		}

		if updatedCount > 0 {
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

func (u *SKUSnapshotUpdater) updateStatusFromResult(req PriceUpdateRequest, result *PriceUpdateResult, finishedAt *time.Time) {
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

func (u *SKUSnapshotUpdater) resolveTargets(ctx context.Context, filter PriceUpdateFilter) ([]domainproduct.Product, error) {
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

func (u *SKUSnapshotUpdater) loadDropshippingDetail(ctx context.Context, product domainproduct.Product, currency string) (*aliexpress.DropshippingProductDetail, error) {
	if product.Market != enum.MarketAliExpress {
		return nil, fmt.Errorf("unsupported market for sku snapshot updater: %s", product.Market)
	}

	return u.aliexpressClient.GetDropshippingProduct(ctx, aliexpress.DropshippingProductRequest{
		ProductID:             product.ExternalProductID,
		ShipToCountry:         "KR",
		TargetCurrency:        currency,
		TargetLanguage:        enum.LanguageForCurrency(currency),
		RemovePersonalBenefit: true,
	})
}

func (u *SKUSnapshotUpdater) recordSKUPrices(ctx context.Context, productID string, detail *aliexpress.DropshippingProductDetail, currency string) (int, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	skus, err := u.productService.FindSKUsByProductID(ctx, productID)
	if err != nil {
		return 0, fmt.Errorf("find skus: %w", err)
	}

	skuMap := make(map[string]domainproduct.SKU, len(skus))
	for _, s := range skus {
		skuMap[strings.TrimSpace(s.ExternalSKUID)] = s
	}

	updatedCount := 0
	for _, dsSku := range detail.SKUs {
		externalSKUID := strings.TrimSpace(dsSku.SKUID)
		dbSku, ok := skuMap[externalSKUID]
		if !ok {
			continue
		}

		price := firstNonEmpty(strings.TrimSpace(dsSku.OfferSalePrice), strings.TrimSpace(dsSku.Price))
		originalPrice := strings.TrimSpace(dsSku.Price)
		if price == "" {
			continue
		}

		changeValue := ""
		lastPrice, _ := u.skuRecorder.GetLatestSKUPrice(ctx, dbSku.ID, currency)
		shouldInsertHistory := lastPrice == ""
		if lastPrice != "" && lastPrice != price {
			shouldInsertHistory = true
			changeValue = calcChange(lastPrice, price)
		}

		if shouldInsertHistory {
			if err := u.skuRecorder.InsertSKUPrice(ctx, dbSku.ID, now, price, currency, changeValue); err != nil {
				log.Printf("sku history %s currency=%s: %v", dbSku.ID, currency, err)
			}
		}
		if err := u.skuRecorder.UpsertSKUSnapshot(ctx, dbSku.ID, today, price, originalPrice, currency); err != nil {
			log.Printf("sku snapshot %s currency=%s: %v", dbSku.ID, currency, err)
			continue
		}
		updatedCount++
	}

	return updatedCount, nil
}

func normalizeSKUSnapshotUpdateRequest(req PriceUpdateRequest) PriceUpdateRequest {
	req = normalizePriceUpdateRequest(req)
	req.JobType = JobTypeSKUSnapshotUpdate
	req.Filter.CollectionSource = strings.TrimSpace(req.Filter.CollectionSource)
	return req
}
