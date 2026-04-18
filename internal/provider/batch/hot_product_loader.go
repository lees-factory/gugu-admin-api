package batch

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/ljj/gugu-admin-api/internal/clients/aliexpress"
	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
	"github.com/ljj/gugu-admin-api/internal/core/enum"
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
	aliasRepo      ProductAliasRepository
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
	aliasRepo ProductAliasRepository,
) *HotProductLoader {
	return &HotProductLoader{
		client:         client,
		productService: productService,
		aliasRepo:      aliasRepo,
	}
}

const (
	hotProductPageSize      = 20
	hotProductMaxPages      = 1
	hotProductShipToCountry = "KR"
)

func (l *HotProductLoader) LoadHotProducts(ctx context.Context, input HotProductLoadInput) (*HotProductLoadResult, error) {
	result := &HotProductLoadResult{}
	currency := enum.SupportedCurrencies[0]
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
			productSaved, skipped, err := l.processHotProductItem(ctx, item)
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

	return result, nil
}

func (l *HotProductLoader) processHotProductItem(
	ctx context.Context,
	item aliexpress.HotProduct,
) (bool, bool, error) {
	viewExternalProductID := strings.TrimSpace(item.ProductID)
	title := strings.TrimSpace(item.ProductTitle)
	imageURL := strings.TrimSpace(item.ProductMainImageURL)
	productURL := strings.TrimSpace(item.ProductDetailURL)
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

	if existing == nil {
		created, err := l.productService.Create(ctx, domainproduct.NewProduct{
			Market:            enum.MarketAliExpress,
			ExternalProductID: lookupProductID,
			OriginalURL:       productURL,
			Title:             title,
			MainImageURL:      imageURL,
			ProductURL:        productURL,
			CollectionSource:  domainproduct.CollectionSourceHotProductQuery,
		})
		if err != nil {
			return false, false, fmt.Errorf("save product %s: %w", lookupProductID, err)
		}
		if err := l.upsertViewAlias(ctx, enum.MarketAliExpress, viewExternalProductID, created.ID); err != nil {
			log.Printf("upsert product alias %s failed: %v", viewExternalProductID, err)
		}
		return true, false, nil
	}

	if err := l.upsertViewAlias(ctx, enum.MarketAliExpress, viewExternalProductID, existing.ID); err != nil {
		log.Printf("upsert product alias %s failed: %v", viewExternalProductID, err)
	}

	return false, true, nil
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
