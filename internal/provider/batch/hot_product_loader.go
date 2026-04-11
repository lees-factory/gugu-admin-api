package batch

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/ljj/gugu-admin-api/internal/clients/aliexpress"
	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
	"github.com/ljj/gugu-admin-api/internal/core/enum"
	"github.com/ljj/gugu-admin-api/internal/storage/dbcore/hotproduct"
	"github.com/ljj/gugu-admin-api/internal/support/id"
)

type HotProductLoadInput struct {
	CategoryIDs  []string `json:"category_ids"`
	Keywords     string   `json:"keywords"`
	Sort         string   `json:"sort"`
	MinSalePrice string   `json:"min_sale_price"`
	MaxSalePrice string   `json:"max_sale_price"`
}

type HotProductLoadResult struct {
	RequestedCount    int `json:"requested_count"`
	ProcessedPages    int `json:"processed_pages"`
	HotProductSaved   int `json:"hot_product_saved"`
	ProductSavedCount int `json:"product_saved_count"`
	SkippedCount      int `json:"skipped_count"`
}

type HotProductLoader struct {
	client         aliexpress.Client
	productService *domainproduct.Service
	hotProductRepo *hotproduct.SQLCRepository
	priceRecorder  PriceHistoryRecorder
	variantWriter  ProductVariantWriter
	aliasRepo      ProductAliasRepository
	idGen          *id.Generator
}

var (
	aliexpressItemPattern  = regexp.MustCompile(`/item/([0-9]+)(?:\.html)?`)
	aliexpressShortPattern = regexp.MustCompile(`/i/([0-9]+)\.html`)
)

type ProductAliasRepository interface {
	FindProductIDByAlias(ctx context.Context, market enum.Market, aliasExternalProductID string) (string, error)
	UpsertViewAlias(ctx context.Context, market enum.Market, aliasExternalProductID, productID string) error
}

func NewHotProductLoader(
	client aliexpress.Client,
	productService *domainproduct.Service,
	hotProductRepo *hotproduct.SQLCRepository,
	priceRecorder PriceHistoryRecorder,
	variantWriter ProductVariantWriter,
	aliasRepo ProductAliasRepository,
	idGen *id.Generator,
) *HotProductLoader {
	return &HotProductLoader{
		client:         client,
		productService: productService,
		hotProductRepo: hotProductRepo,
		priceRecorder:  priceRecorder,
		variantWriter:  variantWriter,
		aliasRepo:      aliasRepo,
		idGen:          idGen,
	}
}

const (
	hotProductPageSize      = 20
	hotProductMaxPages      = 1
	hotProductShipToCountry = "KR"
)

func (l *HotProductLoader) LoadHotProducts(ctx context.Context, input HotProductLoadInput) (*HotProductLoadResult, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	result := &HotProductLoadResult{}
	for _, currency := range enum.SupportedCurrencies {
		lang := enum.LanguageForCurrency(currency)

		for pageOffset := 0; pageOffset < hotProductMaxPages; pageOffset++ {
			pageNo := 1 + pageOffset
			items, err := l.client.QueryHotProducts(ctx, aliexpress.HotProductQueryRequest{
				CategoryIDs:    input.CategoryIDs,
				Keywords:       strings.TrimSpace(input.Keywords),
				PageNo:         pageNo,
				PageSize:       hotProductPageSize,
				Sort:           strings.TrimSpace(input.Sort),
				MinSalePrice:   strings.TrimSpace(input.MinSalePrice),
				MaxSalePrice:   strings.TrimSpace(input.MaxSalePrice),
				ShipToCountry:  hotProductShipToCountry,
				TargetCurrency: currency,
				TargetLanguage: lang,
			})
			if err != nil {
				return nil, fmt.Errorf("query hot products currency %s page %d: %w", currency, pageNo, err)
			}
			if len(items) == 0 {
				break
			}

			result.ProcessedPages++
			result.RequestedCount += len(items)

			for _, item := range items {
				productSaved, skipped, err := l.processHotProductItem(ctx, item, currency, now, today)
				if err != nil {
					return nil, err
				}
				result.HotProductSaved++
				if productSaved {
					result.ProductSavedCount++
				}
				if skipped {
					result.SkippedCount++
				}
			}

			if len(items) < hotProductPageSize {
				break
			}
		}
	}

	return result, nil
}

func (l *HotProductLoader) processHotProductItem(
	ctx context.Context,
	item aliexpress.HotProduct,
	requestedCurrency string,
	now time.Time,
	today time.Time,
) (bool, bool, error) {
	viewExternalProductID := strings.TrimSpace(item.ProductID)
	title := strings.TrimSpace(item.ProductTitle)
	imageURL := strings.TrimSpace(item.ProductMainImageURL)
	productURL := strings.TrimSpace(item.ProductDetailURL)
	price := firstNonEmpty(item.TargetSalePrice, item.SalePrice)
	currency := firstNonEmpty(item.TargetSalePriceCurrency, item.SalePriceCurrency, requestedCurrency)
	originProductID := resolveOriginProductID(viewExternalProductID, productURL)
	lookupProductID := firstNonEmpty(originProductID, viewExternalProductID)

	existing, err := l.findExistingByExternalOrAlias(ctx, enum.MarketAliExpress, lookupProductID)
	if err != nil {
		return false, false, fmt.Errorf("check existing product %s: %w", lookupProductID, err)
	}
	if existing == nil && viewExternalProductID != "" && viewExternalProductID != lookupProductID {
		existing, err = l.findExistingByExternalOrAlias(ctx, enum.MarketAliExpress, viewExternalProductID)
		if err != nil {
			return false, false, fmt.Errorf("check existing product by view id %s: %w", viewExternalProductID, err)
		}
	}

	var productID string
	productSaved := false
	skipped := false

	if existing == nil && currency == enum.SupportedCurrencies[0] {
		created, err := l.productService.Create(ctx, domainproduct.NewProduct{
			Market:            enum.MarketAliExpress,
			ExternalProductID: lookupProductID,
			OriginalURL:       productURL,
			Title:             title,
			MainImageURL:      imageURL,
			CurrentPrice:      price,
			Currency:          currency,
			ProductURL:        productURL,
			CollectionSource:  domainproduct.CollectionSourceHotProductQuery,
		})
		if err != nil {
			return false, false, fmt.Errorf("save product %s: %w", lookupProductID, err)
		}
		productID = created.ID
		productSaved = true
	} else {
		if existing != nil {
			productID = existing.ID
			if currency == enum.SupportedCurrencies[0] {
				skipped = true
			}
		}
		if existing == nil {
			return false, false, nil
		}
	}

	if err := l.recordProductPrice(ctx, productID, price, currency, now, today); err != nil {
		log.Printf("record hot product price %s currency=%s failed: %v", lookupProductID, currency, err)
	}
	if err := l.upsertProductVariant(ctx, productID, title, imageURL, productURL, price, currency, now); err != nil {
		log.Printf("upsert product variant %s currency=%s failed: %v", lookupProductID, currency, err)
	}
	if err := l.upsertViewAlias(ctx, enum.MarketAliExpress, viewExternalProductID, productID); err != nil {
		log.Printf("upsert product alias %s failed: %v", viewExternalProductID, err)
	}

	return productSaved, skipped, nil
}

func (l *HotProductLoader) recordProductPrice(ctx context.Context, productID, price, currency string, now, today time.Time) error {
	if l.priceRecorder == nil || productID == "" || price == "" || currency == "" {
		return nil
	}

	changeValue := ""
	lastPrice, _ := l.priceRecorder.GetLatestProductPrice(ctx, productID, currency)
	shouldInsertHistory := lastPrice == ""
	if lastPrice != "" && lastPrice != price {
		shouldInsertHistory = true
		changeValue = calcChange(lastPrice, price)
	}

	if shouldInsertHistory {
		if err := l.priceRecorder.InsertProductPrice(ctx, productID, now, price, currency, changeValue); err != nil {
			return err
		}
	}
	if err := l.priceRecorder.UpsertProductSnapshot(ctx, productID, today, price, currency); err != nil {
		return err
	}

	return nil
}

func (l *HotProductLoader) upsertProductVariant(ctx context.Context, productID, title, imageURL, productURL, price, currency string, collectedAt time.Time) error {
	if l.variantWriter == nil || productID == "" || currency == "" {
		return nil
	}

	return l.variantWriter.UpsertProductVariant(
		ctx,
		productID,
		enum.LanguageForCurrency(currency),
		currency,
		title,
		imageURL,
		productURL,
		price,
		collectedAt,
	)
}

func (l *HotProductLoader) findExistingByExternalOrAlias(ctx context.Context, market enum.Market, externalProductID string) (*domainproduct.Product, error) {
	existing, err := l.productService.FindByMarketAndExternalProductID(ctx, market, externalProductID)
	if err != nil || existing != nil || l.aliasRepo == nil {
		return existing, err
	}

	productID, err := l.aliasRepo.FindProductIDByAlias(ctx, market, externalProductID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(productID) == "" {
		return nil, nil
	}

	resolved, err := l.productService.FindByID(ctx, productID)
	if err != nil {
		log.Printf("alias resolved product is missing: market=%s alias=%s product_id=%s err=%v", market, externalProductID, productID, err)
		return nil, nil
	}
	return resolved, nil
}

func (l *HotProductLoader) upsertViewAlias(ctx context.Context, market enum.Market, aliasExternalProductID, productID string) error {
	if l.aliasRepo == nil {
		return nil
	}
	aliasExternalProductID = strings.TrimSpace(aliasExternalProductID)
	productID = strings.TrimSpace(productID)
	if aliasExternalProductID == "" || productID == "" {
		return nil
	}

	return l.aliasRepo.UpsertViewAlias(ctx, market, aliasExternalProductID, productID)
}

func resolveOriginProductID(viewExternalProductID, productURL string) string {
	if id := extractAliExpressProductIDFromURL(productURL); id != "" {
		return id
	}
	return strings.TrimSpace(viewExternalProductID)
}

func extractAliExpressProductIDFromURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	if match := aliexpressItemPattern.FindStringSubmatch(rawURL); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	if match := aliexpressShortPattern.FindStringSubmatch(rawURL); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}
