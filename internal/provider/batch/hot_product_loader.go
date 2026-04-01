package batch

import (
	"context"
	"fmt"
	"log"
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
	idGen          *id.Generator
}

func NewHotProductLoader(
	client aliexpress.Client,
	productService *domainproduct.Service,
	hotProductRepo *hotproduct.SQLCRepository,
	idGen *id.Generator,
) *HotProductLoader {
	return &HotProductLoader{
		client:         client,
		productService: productService,
		hotProductRepo: hotProductRepo,
		idGen:          idGen,
	}
}

const (
	hotProductPageSize       = 20
	hotProductMaxPages       = 1
	hotProductTargetCurrency = "KRW"
	hotProductTargetLanguage = "KO"
	hotProductShipToCountry  = "KR"
)

func (l *HotProductLoader) LoadHotProducts(ctx context.Context, input HotProductLoadInput) (*HotProductLoadResult, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	result := &HotProductLoadResult{}
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
			TargetCurrency: hotProductTargetCurrency,
			TargetLanguage: hotProductTargetLanguage,
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
			promotionLink := strings.TrimSpace(item.PromotionLink)
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
				PromotionLink:     promotionLink,
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

			_, err = l.productService.Create(ctx, domainproduct.NewProduct{
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
		}

		if len(items) < hotProductPageSize {
			break
		}
	}

	return result, nil
}
