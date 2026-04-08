package batch

import (
	"context"
	"log"
	"time"
)

type HotProductScheduler struct {
	loader      *HotProductLoader
	skuEnricher *SKUEnricher
	snapshotter *SKUSnapshotUpdater
	interval    time.Duration
	input       HotProductLoadInput
}

func NewHotProductScheduler(
	loader *HotProductLoader,
	skuEnricher *SKUEnricher,
	snapshotter *SKUSnapshotUpdater,
	interval time.Duration,
) *HotProductScheduler {
	return &HotProductScheduler{
		loader:      loader,
		skuEnricher: skuEnricher,
		snapshotter: snapshotter,
		interval:    interval,
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

	req := PriceUpdateRequest{
		TriggerType: TriggerTypeScheduled,
		RequestedBy: "internal-scheduler",
		Metadata: map[string]string{
			"runner":   "internal-scheduler",
			"pipeline": "hot-product-scheduler",
		},
	}

	status, err := s.snapshotter.Preview(ctx, req)
	if err != nil {
		log.Printf("hot product scheduler snapshot preview failed: %v", err)
		return
	}

	log.Printf("hot product scheduler snapshot triggered: total=%d", status.TotalCount)

	resultStatus, err := s.snapshotter.Run(ctx, req)
	if err != nil {
		log.Printf("hot product scheduler snapshot run failed: %v", err)
		return
	}

	log.Printf("hot product scheduler snapshot completed: total=%d success=%d fail=%d skipped=%d",
		resultStatus.TotalCount, resultStatus.SuccessCount, resultStatus.FailCount, resultStatus.SkippedCount)
}
