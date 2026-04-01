package batch

import (
	"context"
	"log"
	"time"
)

type HotProductScheduler struct {
	loader      *HotProductLoader
	skuEnricher *SKUEnricher
	interval    time.Duration
	input       HotProductLoadInput
}

func NewHotProductScheduler(loader *HotProductLoader, skuEnricher *SKUEnricher, interval time.Duration) *HotProductScheduler {
	return &HotProductScheduler{
		loader:      loader,
		skuEnricher: skuEnricher,
		interval:    interval,
		input:       HotProductLoadInput{},
	}
}

func (s *HotProductScheduler) Start(ctx context.Context) {
	if s == nil || s.loader == nil || s.interval <= 0 {
		return
	}

	ticker := time.NewTicker(s.interval)

	go func() {
		defer ticker.Stop()

		log.Printf("hot product scheduler started: interval=%s", s.interval)

		for {
			select {
			case <-ctx.Done():
				log.Printf("hot product scheduler stopped: %v", ctx.Err())
				return
			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

func (s *HotProductScheduler) runOnce(ctx context.Context) {
	result, err := s.loader.LoadHotProducts(ctx, s.input)
	if err != nil {
		log.Printf("hot product scheduler load failed: %v", err)
		return
	}

	log.Printf("hot product scheduler load completed: requested=%d hot_saved=%d product_saved=%d skipped=%d",
		result.RequestedCount, result.HotProductSaved, result.ProductSavedCount, result.SkippedCount)

	if s.skuEnricher == nil {
		return
	}

	enrichResult, err := s.skuEnricher.EnrichHotProducts(ctx)
	if err != nil {
		log.Printf("hot product scheduler enrich failed: %v", err)
		return
	}

	log.Printf("hot product scheduler enrich completed: total=%d success=%d fail=%d skus_added=%d",
		enrichResult.TotalProducts, enrichResult.SuccessCount, enrichResult.FailCount, enrichResult.TotalSKUsAdded)
}
