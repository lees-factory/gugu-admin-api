package batch

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/ljj/gugu-admin-api/internal/clients/aliexpress"
	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
	"github.com/ljj/gugu-admin-api/internal/core/enum"
)

type SKUPriceRecorder interface {
	InsertSKUPrice(ctx context.Context, skuID string, recordedAt time.Time, price, currency, changeValue string) error
	GetLatestSKUPrice(ctx context.Context, skuID, currency string) (string, error)
	UpsertSKUSnapshot(ctx context.Context, skuID string, snapshotDate time.Time, price, originalPrice, currency string) error
}

type SKUEnricher struct {
	productService   *domainproduct.Service
	aliexpressClient aliexpress.Client
	skuRecorder      SKUPriceRecorder
	minDelay         time.Duration
	maxDelay         time.Duration
}

type EnrichResult struct {
	TotalProducts  int `json:"total_products"`
	SuccessCount   int `json:"success_count"`
	FailCount      int `json:"fail_count"`
	TotalSKUsAdded int `json:"total_skus_added"`
}

func NewSKUEnricher(
	productService *domainproduct.Service,
	aliexpressClient aliexpress.Client,
	skuRecorder SKUPriceRecorder,
	minDelay time.Duration,
	maxDelay time.Duration,
) *SKUEnricher {
	return &SKUEnricher{
		productService:   productService,
		aliexpressClient: aliexpressClient,
		skuRecorder:      skuRecorder,
		minDelay:         minDelay,
		maxDelay:         maxDelay,
	}
}

func (e *SKUEnricher) EnrichHotProducts(ctx context.Context) (*EnrichResult, error) {
	products, err := e.productService.ListByCollectionSource(ctx, domainproduct.CollectionSourceHotProductQuery)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	var targets []domainproduct.Product
	for _, p := range products {
		count, err := e.productService.CountSKUsByProductID(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("count skus for %s: %w", p.ID, err)
		}
		if count == 0 {
			targets = append(targets, p)
		}
	}

	return e.enrichProducts(ctx, targets)
}

func (e *SKUEnricher) EnrichAll(ctx context.Context) (*EnrichResult, error) {
	products, err := e.productService.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all products: %w", err)
	}

	return e.enrichProducts(ctx, products)
}

func (e *SKUEnricher) enrichProducts(ctx context.Context, products []domainproduct.Product) (*EnrichResult, error) {
	result := &EnrichResult{TotalProducts: len(products)}

	for i, p := range products {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		log.Printf("[%d/%d] enriching product: %s (external: %s)", i+1, len(products), p.ID, p.ExternalProductID)

		added, err := e.enrichSingleFromDropshipping(ctx, p)
		if err != nil {
			log.Printf("[%d/%d] FAILED: %v", i+1, len(products), err)
			result.FailCount++
			continue
		}

		log.Printf("[%d/%d] SUCCESS: %d SKUs upserted", i+1, len(products), added)
		result.SuccessCount++
		result.TotalSKUsAdded += added

		if i < len(products)-1 {
			e.randomDelay()
		}
	}

	return result, nil
}

func (e *SKUEnricher) enrichSingleFromDropshipping(ctx context.Context, p domainproduct.Product) (int, error) {
	if e.aliexpressClient == nil {
		return 0, fmt.Errorf("aliexpress client is not configured")
	}

	totalUpserted := 0

	for ci, currency := range enum.SupportedCurrencies {
		lang := enum.LanguageForCurrency(currency)

		detail, err := e.aliexpressClient.GetDropshippingProduct(ctx, aliexpress.DropshippingProductRequest{
			ProductID:             p.ExternalProductID,
			ShipToCountry:         "KR",
			TargetCurrency:        currency,
			TargetLanguage:        lang,
			RemovePersonalBenefit: true,
		})
		if err != nil {
			log.Printf("dropshipping %s currency=%s: %v", p.ExternalProductID, currency, err)
			continue
		}

		// 첫 번째 통화(KRW)에서만 SKU upsert (테이블 데이터는 대표 통화 기준)
		if ci == 0 {
			skuInputs := make([]domainproduct.NewSKU, len(detail.SKUs))
			for i, sku := range detail.SKUs {
				skuInputs[i] = domainproduct.NewSKU{
					ExternalSKUID: strings.TrimSpace(sku.SKUID),
					OriginSKUID:   strings.TrimSpace(sku.OriginSKUID),
					SKUName:       strings.TrimSpace(sku.SKUName),
					Color:         strings.TrimSpace(sku.Color),
					Size:          strings.TrimSpace(sku.Size),
					Price:         firstNonEmpty(strings.TrimSpace(sku.OfferSalePrice), strings.TrimSpace(sku.Price)),
					OriginalPrice: strings.TrimSpace(sku.Price),
					Currency:      firstNonEmpty(strings.TrimSpace(sku.CurrencyCode), strings.TrimSpace(detail.CurrencyCode)),
					ImageURL:      strings.TrimSpace(sku.ImageURL),
					SKUProperties: strings.TrimSpace(sku.SKUAttr),
				}
			}

			upserted, err := e.productService.EnrichSKUs(ctx, p.ID, skuInputs)
			if err != nil {
				return 0, fmt.Errorf("enrich skus %s: %w", p.ExternalProductID, err)
			}
			totalUpserted = upserted
		}

		// 모든 통화에서 sku_price_history 기록
		e.recordSKUPrices(ctx, p.ID, detail, currency)

		if ci < len(enum.SupportedCurrencies)-1 {
			e.randomDelay()
		}
	}

	return totalUpserted, nil
}

func (e *SKUEnricher) recordSKUPrices(ctx context.Context, productID string, detail *aliexpress.DropshippingProductDetail, currency string) {
	if e.skuRecorder == nil {
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	skus, err := e.productService.FindSKUsByProductID(ctx, productID)
	if err != nil {
		log.Printf("find skus for history %s: %v", productID, err)
		return
	}

	skuMap := make(map[string]domainproduct.SKU)
	for _, s := range skus {
		skuMap[s.ExternalSKUID] = s
	}

	for _, dsSku := range detail.SKUs {
		externalSKUID := strings.TrimSpace(dsSku.SKUID)
		dbSku, ok := skuMap[externalSKUID]
		if !ok {
			continue
		}

		price := firstNonEmpty(strings.TrimSpace(dsSku.OfferSalePrice), strings.TrimSpace(dsSku.Price))
		originalPrice := strings.TrimSpace(dsSku.Price)

		changeValue := ""
		lastPrice, _ := e.skuRecorder.GetLatestSKUPrice(ctx, dbSku.ID, currency)
		if lastPrice != "" && lastPrice != price {
			changeValue = calcChange(lastPrice, price)
		}

		if err := e.skuRecorder.InsertSKUPrice(ctx, dbSku.ID, now, price, currency, changeValue); err != nil {
			log.Printf("sku history %s currency=%s: %v", dbSku.ID, currency, err)
		}
		if err := e.skuRecorder.UpsertSKUSnapshot(ctx, dbSku.ID, today, price, originalPrice, currency); err != nil {
			log.Printf("sku snapshot %s currency=%s: %v", dbSku.ID, currency, err)
		}
	}
}

func (e *SKUEnricher) randomDelay() {
	diff := e.maxDelay - e.minDelay
	delay := e.minDelay + time.Duration(rand.Int64N(int64(diff)))
	log.Printf("waiting %s before next request...", delay.Round(time.Second))
	time.Sleep(delay)
}

func buildAliExpressURL(externalProductID string) string {
	return fmt.Sprintf("https://ko.aliexpress.com/item/%s.html", externalProductID)
}
