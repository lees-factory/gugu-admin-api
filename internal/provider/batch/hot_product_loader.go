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
	"github.com/ljj/gugu-admin-api/internal/storage/dbcore/hotproduct"
	"github.com/ljj/gugu-admin-api/internal/support/id"
)

type HotProductLoadInput struct {
	CategoryIDs    []string `json:"category_ids"`
	Keywords       string   `json:"keywords"`
	PageNo         int      `json:"page_no"`
	PageSize       int      `json:"page_size"`
	MaxPages       int      `json:"max_pages"`
	Sort           string   `json:"sort"`
	MinSalePrice   string   `json:"min_sale_price"`
	MaxSalePrice   string   `json:"max_sale_price"`
	ShipToCountry  string   `json:"ship_to_country"`
	TargetCurrency string   `json:"target_currency"`
	TargetLanguage string   `json:"target_language"`
}

type HotProductLoadResult struct {
	RequestedCount    int `json:"requested_count"`
	ProcessedPages    int `json:"processed_pages"`
	HotProductSaved   int `json:"hot_product_saved"`
	ProductSavedCount int `json:"product_saved_count"`
	SKUSavedCount     int `json:"sku_saved_count"`
	SkippedCount      int `json:"skipped_count"`
	SKUFailCount      int `json:"sku_fail_count"`
}

type HotProductLoader struct {
	client         aliexpress.Client
	productService *domainproduct.Service
	hotProductRepo *hotproduct.SQLCRepository
	skuRecorder    SKUPriceRecorder
	idGen          *id.Generator
	skuDelay       time.Duration
}

func NewHotProductLoader(
	client aliexpress.Client,
	productService *domainproduct.Service,
	hotProductRepo *hotproduct.SQLCRepository,
	skuRecorder SKUPriceRecorder,
	idGen *id.Generator,
) *HotProductLoader {
	return &HotProductLoader{
		client:         client,
		productService: productService,
		hotProductRepo: hotProductRepo,
		skuRecorder:    skuRecorder,
		idGen:          idGen,
		skuDelay:       3 * time.Second,
	}
}

func (l *HotProductLoader) LoadHotProducts(ctx context.Context, input HotProductLoadInput) (*HotProductLoadResult, error) {
	if input.PageNo <= 0 {
		input.PageNo = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 20
	}
	if input.PageSize > 50 {
		input.PageSize = 50
	}
	if input.MaxPages <= 0 {
		input.MaxPages = 100
	}
	if input.TargetCurrency == "" {
		input.TargetCurrency = "KRW"
	}
	if input.TargetLanguage == "" {
		input.TargetLanguage = "KO"
	}
	if input.ShipToCountry == "" {
		input.ShipToCountry = "KR"
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	result := &HotProductLoadResult{}
	for pageOffset := 0; pageOffset < input.MaxPages; pageOffset++ {
		pageNo := input.PageNo + pageOffset
		items, err := l.client.QueryHotProducts(ctx, aliexpress.HotProductQueryRequest{
			CategoryIDs:    input.CategoryIDs,
			Keywords:       strings.TrimSpace(input.Keywords),
			PageNo:         pageNo,
			PageSize:       input.PageSize,
			Sort:           strings.TrimSpace(input.Sort),
			MinSalePrice:   strings.TrimSpace(input.MinSalePrice),
			MaxSalePrice:   strings.TrimSpace(input.MaxSalePrice),
			ShipToCountry:  strings.TrimSpace(input.ShipToCountry),
			TargetCurrency: strings.TrimSpace(input.TargetCurrency),
			TargetLanguage: strings.TrimSpace(input.TargetLanguage),
		})
		if err != nil {
			return nil, fmt.Errorf("query hot products page %d: %w", pageNo, err)
		}
		if len(items) == 0 {
			break
		}

		result.ProcessedPages++
		result.RequestedCount += len(items)

		for _, item := range items {
			externalProductID := strings.TrimSpace(item.ProductID)
			title := strings.TrimSpace(item.ProductTitle)
			imageURL := strings.TrimSpace(item.ProductMainImageURL)
			productURL := strings.TrimSpace(item.ProductDetailURL)
			price := firstNonEmpty(item.TargetSalePrice, item.SalePrice)
			currency := firstNonEmpty(item.TargetSalePriceCurrency, item.SalePriceCurrency)

			hotID, err := l.idGen.New()
			if err != nil {
				return nil, fmt.Errorf("generate hot product id: %w", err)
			}
			if err := l.hotProductRepo.Insert(ctx, hotproduct.HotProductRow{
				ID:                hotID,
				ExternalProductID: externalProductID,
				Title:             title,
				ImageURL:          imageURL,
				ProductURL:        productURL,
				SalePrice:         price,
				Currency:          currency,
				CollectedDate:     today,
				CreatedAt:         now,
			}); err != nil {
				log.Printf("insert hot_product %s failed: %v", externalProductID, err)
				continue
			}
			result.HotProductSaved++

			existing, err := l.productService.FindByMarketAndExternalProductID(ctx, enum.MarketAliExpress, externalProductID)
			if err != nil {
				return nil, fmt.Errorf("check existing product %s: %w", externalProductID, err)
			}
			if existing != nil {
				result.SkippedCount++
				continue
			}

			created, err := l.productService.Create(ctx, domainproduct.NewProduct{
				Market:            enum.MarketAliExpress,
				ExternalProductID: externalProductID,
				OriginalURL:       productURL,
				Title:             title,
				MainImageURL:      imageURL,
				CurrentPrice:      price,
				Currency:          currency,
				ProductURL:        productURL,
				CollectionSource:  domainproduct.CollectionSourceHotProductQuery,
			})
			if err != nil {
				return nil, fmt.Errorf("save product %s: %w", externalProductID, err)
			}
			result.ProductSavedCount++

			l.randomDelay()
			skuCount, skuErr := l.loadSKUs(ctx, created.ID, externalProductID)
			if skuErr != nil {
				log.Printf("sku load failed for %s: %v", externalProductID, skuErr)
				result.SKUFailCount++
			} else {
				result.SKUSavedCount += skuCount
			}
		}

		if len(items) < input.PageSize {
			break
		}
	}

	return result, nil
}

func (l *HotProductLoader) loadSKUs(ctx context.Context, productID, externalProductID string) (int, error) {
	totalUpserted := 0

	for ci, currency := range enum.SupportedCurrencies {
		lang := enum.LanguageForCurrency(currency)

		detail, err := l.client.GetDropshippingProduct(ctx, aliexpress.DropshippingProductRequest{
			ProductID:             externalProductID,
			ShipToCountry:         "KR",
			TargetCurrency:        currency,
			TargetLanguage:        lang,
			RemovePersonalBenefit: true,
		})
		if err != nil {
			log.Printf("dropshipping %s currency=%s: %v", externalProductID, currency, err)
			continue
		}

		// 첫 번째 통화(KRW)에서만 SKU upsert
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

			upserted, err := l.productService.EnrichSKUs(ctx, productID, skuInputs)
			if err != nil {
				return 0, fmt.Errorf("enrich skus %s: %w", externalProductID, err)
			}
			totalUpserted = upserted
		}

		// 모든 통화에서 sku_price_history 기록
		l.recordSKUPrices(ctx, productID, detail, currency)

		if ci < len(enum.SupportedCurrencies)-1 {
			l.randomDelay()
		}
	}

	return totalUpserted, nil
}

func (l *HotProductLoader) recordSKUPrices(ctx context.Context, productID string, detail *aliexpress.DropshippingProductDetail, currency string) {
	if l.skuRecorder == nil {
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	skus, err := l.productService.FindSKUsByProductID(ctx, productID)
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
		lastPrice, _ := l.skuRecorder.GetLatestSKUPrice(ctx, dbSku.ID, currency)
		if lastPrice != "" && lastPrice != price {
			changeValue = calcChange(lastPrice, price)
		}

		if err := l.skuRecorder.InsertSKUPrice(ctx, dbSku.ID, now, price, currency, changeValue); err != nil {
			log.Printf("sku history %s currency=%s: %v", dbSku.ID, currency, err)
		}
		if err := l.skuRecorder.UpsertSKUSnapshot(ctx, dbSku.ID, today, price, originalPrice, currency); err != nil {
			log.Printf("sku snapshot %s currency=%s: %v", dbSku.ID, currency, err)
		}
	}
}

func (l *HotProductLoader) randomDelay() {
	base := l.skuDelay
	jitter := time.Duration(rand.Int64N(int64(base)))
	delay := base + jitter
	time.Sleep(delay)
}
