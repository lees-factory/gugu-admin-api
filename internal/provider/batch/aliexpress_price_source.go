package batch

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ljj/gugu-admin-api/internal/clients/aliexpress"
	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
	"github.com/ljj/gugu-admin-api/internal/core/enum"
)

type ProductPriceSource interface {
	Load(ctx context.Context, product domainproduct.Product, currency string) (*ProductPricePayload, error)
}

type ProductPricePayload struct {
	ProductID    string `json:"product_id"`
	Title        string `json:"title"`
	CurrentPrice string `json:"current_price"`
	Currency     string `json:"currency"`
	ProductURL   string `json:"product_url"`
	MainImageURL string `json:"main_image_url"`
	PriceSource  string `json:"price_source"`
}

type AliExpressPriceSource struct {
	client          aliexpress.Client
	minRequestDelay time.Duration
	mu              sync.Mutex
	lastRequestedAt time.Time
}

func NewAliExpressPriceSource(client aliexpress.Client, minRequestDelay time.Duration) *AliExpressPriceSource {
	return &AliExpressPriceSource{
		client:          client,
		minRequestDelay: minRequestDelay,
	}
}

func (s *AliExpressPriceSource) Load(ctx context.Context, product domainproduct.Product, currency string) (*ProductPricePayload, error) {
	if product.Market != enum.MarketAliExpress {
		return nil, fmt.Errorf("unsupported market for ali express price source: %s", product.Market)
	}

	lang := enum.LanguageForCurrency(currency)

	if err := s.waitTurn(ctx); err != nil {
		return nil, err
	}

	affiliatePayload, err := s.loadFromAffiliate(ctx, product, currency, lang)
	if err == nil && affiliatePayload != nil {
		return affiliatePayload, nil
	}

	if err := s.waitTurn(ctx); err != nil {
		return nil, err
	}

	dropshippingPayload, dsErr := s.loadFromDropshipping(ctx, product, currency, lang)
	if dsErr != nil {
		if err != nil {
			return nil, fmt.Errorf("affiliate detail failed: %v; dropshipping detail failed: %w", err, dsErr)
		}
		return nil, fmt.Errorf("dropshipping detail failed: %w", dsErr)
	}

	return dropshippingPayload, nil
}

func (s *AliExpressPriceSource) waitTurn(ctx context.Context) error {
	if s.minRequestDelay <= 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if s.lastRequestedAt.IsZero() {
		s.lastRequestedAt = now
		return nil
	}

	wait := s.minRequestDelay - now.Sub(s.lastRequestedAt)
	if wait <= 0 {
		s.lastRequestedAt = now
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		s.lastRequestedAt = time.Now()
		return nil
	}
}

func (s *AliExpressPriceSource) loadFromAffiliate(ctx context.Context, product domainproduct.Product, currency, lang string) (*ProductPricePayload, error) {
	items, err := s.client.GetAffiliateProductDetails(ctx, aliexpress.AffiliateProductDetailRequest{
		ProductIDs:     []string{product.ExternalProductID},
		TargetCurrency: currency,
		TargetLanguage: lang,
		Country:        "KR",
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("affiliate product detail is empty")
	}

	item := items[0]
	price := firstNonEmpty(item.TargetSalePrice, item.SalePrice)
	respCurrency := firstNonEmpty(item.TargetSalePriceCurrency, item.SalePriceCurrency, currency)
	if price == "" {
		return nil, fmt.Errorf("affiliate product detail price is empty")
	}

	return &ProductPricePayload{
		ProductID:    product.ID,
		Title:        firstNonEmpty(item.ProductTitle, product.Title),
		CurrentPrice: price,
		Currency:     respCurrency,
		ProductURL:   firstNonEmpty(item.ProductDetailURL, product.ProductURL, buildAliExpressURL(product.ExternalProductID)),
		MainImageURL: firstNonEmpty(item.ProductMainImageURL, product.MainImageURL),
		PriceSource:  "ALIEXPRESS_AFFILIATE_PRODUCTDETAIL",
	}, nil
}

func (s *AliExpressPriceSource) loadFromDropshipping(ctx context.Context, product domainproduct.Product, currency, lang string) (*ProductPricePayload, error) {
	item, err := s.client.GetDropshippingProduct(ctx, aliexpress.DropshippingProductRequest{
		ProductID:             product.ExternalProductID,
		ShipToCountry:         "KR",
		TargetCurrency:        currency,
		TargetLanguage:        lang,
		RemovePersonalBenefit: true,
	})
	if err != nil {
		return nil, err
	}

	price := ""
	if len(item.SKUs) > 0 {
		price = firstNonEmpty(item.SKUs[0].OfferSalePrice, item.SKUs[0].Price)
	}
	if price == "" {
		return nil, fmt.Errorf("dropshipping product price is empty")
	}

	return &ProductPricePayload{
		ProductID:    product.ID,
		Title:        firstNonEmpty(item.Subject, product.Title),
		CurrentPrice: price,
		Currency:     firstNonEmpty(item.CurrencyCode, product.Currency),
		ProductURL:   firstNonEmpty(product.ProductURL, buildAliExpressURL(product.ExternalProductID)),
		MainImageURL: firstNonEmpty(firstImage(item.ImageURLs), product.MainImageURL),
		PriceSource:  "ALIEXPRESS_DROPSHIPPING_PRODUCT",
	}, nil
}

func firstImage(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
