package batch

import (
	"context"
	"log"
	"slices"
	"time"

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

	phaseJobs := []struct {
		currency    string
		targetGroup TargetGroup
	}{
		{currency: "KRW", targetGroup: TargetGroupAll},
		{currency: "USD", targetGroup: TargetGroupHotProducts},
		{currency: "USD", targetGroup: TargetGroupTracked},
	}

	for i, job := range phaseJobs {
		if !slices.Contains(enum.SupportedCurrencies, job.currency) {
			continue
		}
		if err := s.runSnapshotForCurrencyAndGroup(ctx, job.currency, job.targetGroup); err != nil {
			log.Printf("hot product scheduler snapshot currency=%s target_group=%s failed: %v", job.currency, job.targetGroup, err)
		}
		if i < len(phaseJobs)-1 {
			s.waitStagger(ctx)
		}
	}
}

func (s *HotProductScheduler) runSnapshotForCurrencyAndGroup(ctx context.Context, currency string, targetGroup TargetGroup) error {
	req := PriceUpdateRequest{
		TriggerType: TriggerTypeScheduled,
		RequestedBy: "internal-scheduler",
		Filter: PriceUpdateFilter{
			Currencies:  []string{currency},
			TargetGroup: targetGroup,
		},
		Metadata: map[string]string{
			"runner":       "internal-scheduler",
			"pipeline":     "hot-product-scheduler",
			"currency":     currency,
			"target_group": string(targetGroup),
		},
	}

	status, err := s.snapshotter.Preview(ctx, req)
	if err != nil {
		return err
	}
	log.Printf("hot product scheduler snapshot triggered: currency=%s target_group=%s total=%d",
		currency, targetGroup, status.TotalCount)

	resultStatus, err := s.snapshotter.Run(ctx, req)
	if err != nil {
		return err
	}
	log.Printf("hot product scheduler snapshot completed: currency=%s target_group=%s total=%d success=%d fail=%d skipped=%d",
		currency, targetGroup, resultStatus.TotalCount, resultStatus.SuccessCount, resultStatus.FailCount, resultStatus.SkippedCount)
	if currency == "USD" && targetGroup == TargetGroupTracked {
		log.Printf("hot product scheduler phase1 sequence done: KRW(ALL) -> USD(HOT_PRODUCTS) -> USD(TRACKED)")
	}
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
