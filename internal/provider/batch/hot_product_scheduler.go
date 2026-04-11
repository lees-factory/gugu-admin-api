package batch

import (
	"context"
	"log"
	"time"

	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
	"github.com/ljj/gugu-admin-api/internal/core/enum"
)

type HotProductScheduler struct {
	loader      *HotProductLoader
	skuEnricher *SKUEnricher
	snapshotter *SKUSnapshotUpdater
	interval    time.Duration
	stagger     time.Duration
	input       HotProductLoadInput
}

func NewHotProductScheduler(
	loader *HotProductLoader,
	skuEnricher *SKUEnricher,
	snapshotter *SKUSnapshotUpdater,
	interval time.Duration,
	stagger time.Duration,
) *HotProductScheduler {
	return &HotProductScheduler{
		loader:      loader,
		skuEnricher: skuEnricher,
		snapshotter: snapshotter,
		interval:    interval,
		stagger:     stagger,
		input:       HotProductLoadInput{},
	}
}

func (s *HotProductScheduler) Start(ctx context.Context) {
	if s == nil || s.loader == nil || s.interval <= 0 {
		return
	}

	startScheduleLoop(ctx, "hot product scheduler", s.interval, shouldAlignToMidnight(s.interval), s.runOnce)
}

func (s *HotProductScheduler) runOnce(ctx context.Context) {
	result, err := s.loader.LoadHotProducts(ctx, s.input)
	if err != nil {
		log.Printf("hot product scheduler load failed: %v", err)
		return
	}

	log.Printf("hot product scheduler load completed: requested=%d hot_saved=%d product_saved=%d skipped=%d",
		result.RequestedCount, result.HotProductSaved, result.ProductSavedCount, result.SkippedCount)

	if s.skuEnricher != nil {
		enrichResult, err := s.skuEnricher.EnrichHotProducts(ctx)
		if err != nil {
			log.Printf("hot product scheduler enrich failed: %v", err)
			return
		}

		log.Printf("hot product scheduler enrich completed: total=%d success=%d fail=%d skus_added=%d",
			enrichResult.TotalProducts, enrichResult.SuccessCount, enrichResult.FailCount, enrichResult.TotalSKUsAdded)
	}

	if s.snapshotter == nil {
		return
	}

	currencies := []string{enum.SupportedCurrencies[0], enum.SupportedCurrencies[1]}
	for i, currency := range currencies {
		if err := s.runSnapshotForCurrency(ctx, currency); err != nil {
			log.Printf("hot product scheduler snapshot currency=%s failed: %v", currency, err)
		}
		if i < len(currencies)-1 {
			s.waitStagger(ctx)
		}
	}
}

func (s *HotProductScheduler) runSnapshotForCurrency(ctx context.Context, currency string) error {
	req := PriceUpdateRequest{
		TriggerType: TriggerTypeScheduled,
		RequestedBy: "internal-scheduler",
		Filter: PriceUpdateFilter{
			CollectionSource: domainproduct.CollectionSourceHotProductQuery,
			Currencies:       []string{currency},
		},
		Metadata: map[string]string{
			"runner":   "internal-scheduler",
			"pipeline": "hot-product-scheduler",
			"currency": currency,
		},
	}

	status, err := s.snapshotter.Preview(ctx, req)
	if err != nil {
		return err
	}
	log.Printf("hot product scheduler snapshot triggered: currency=%s total=%d", currency, status.TotalCount)

	resultStatus, err := s.snapshotter.Run(ctx, req)
	if err != nil {
		return err
	}
	log.Printf("hot product scheduler snapshot completed: currency=%s total=%d success=%d fail=%d skipped=%d",
		currency, resultStatus.TotalCount, resultStatus.SuccessCount, resultStatus.FailCount, resultStatus.SkippedCount)
	return nil
}

func (s *HotProductScheduler) waitStagger(ctx context.Context) {
	if s.stagger <= 0 {
		return
	}
	timer := time.NewTimer(s.stagger)
	defer timer.Stop()

	log.Printf("hot product scheduler stagger wait: %s", s.stagger)
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
