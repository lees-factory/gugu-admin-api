package product

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ljj/gugu-admin-api/internal/core/enum"
	coreerror "github.com/ljj/gugu-admin-api/internal/core/error"
)

type Service struct {
	finder        Finder
	writer        Writer
	skuRepository SKURepository
	idGenerator   IDGenerator
	clock         Clock
}

func NewService(
	finder Finder,
	writer Writer,
	skuRepository SKURepository,
	idGenerator IDGenerator,
	clock Clock,
) *Service {
	return &Service{
		finder:        finder,
		writer:        writer,
		skuRepository: skuRepository,
		idGenerator:   idGenerator,
		clock:         clock,
	}
}

func (s *Service) FindByID(ctx context.Context, productID string) (*Product, error) {
	found, err := s.finder.FindByID(ctx, strings.TrimSpace(productID))
	if err != nil {
		return nil, fmt.Errorf("find product: %w", err)
	}
	if found == nil {
		return nil, coreerror.New(coreerror.ResourceNotFound)
	}
	return found, nil
}

func (s *Service) FindByMarketAndExternalProductID(ctx context.Context, market enum.Market, externalProductID string) (*Product, error) {
	return s.finder.FindByMarketAndExternalProductID(ctx, market, strings.TrimSpace(externalProductID))
}

func (s *Service) FindByIDs(ctx context.Context, productIDs []string) ([]Product, error) {
	trimmed := make([]string, 0, len(productIDs))
	for _, productID := range productIDs {
		productID = strings.TrimSpace(productID)
		if productID == "" {
			continue
		}
		trimmed = append(trimmed, productID)
	}
	return s.finder.FindByIDs(ctx, trimmed)
}

func (s *Service) ListByMarket(ctx context.Context, market enum.Market) ([]Product, error) {
	return s.finder.ListByMarket(ctx, market)
}

func (s *Service) ListByCollectionSource(ctx context.Context, collectionSource string) ([]Product, error) {
	return s.finder.ListByCollectionSource(ctx, strings.TrimSpace(collectionSource))
}

func (s *Service) ListAllLocalized(ctx context.Context, language string) ([]LocalizedProduct, error) {
	return s.finder.ListAllLocalized(ctx, enum.NormalizeLanguage(language))
}

func (s *Service) ListByCollectionSourceLocalized(ctx context.Context, collectionSource, language string) ([]LocalizedProduct, error) {
	return s.finder.ListByCollectionSourceLocalized(ctx, strings.TrimSpace(collectionSource), enum.NormalizeLanguage(language))
}

func (s *Service) ListPriceUpdateCandidates(ctx context.Context, filter PriceUpdateCandidateFilter) ([]Product, error) {
	filter.CollectionSource = strings.TrimSpace(filter.CollectionSource)
	return s.finder.ListPriceUpdateCandidates(ctx, filter)
}

func (s *Service) ListAll(ctx context.Context) ([]Product, error) {
	return s.finder.ListAll(ctx)
}

func (s *Service) FindSKUsByProductID(ctx context.Context, productID string) ([]SKU, error) {
	return s.skuRepository.FindByProductID(ctx, strings.TrimSpace(productID))
}

func (s *Service) CountSKUsByProductID(ctx context.Context, productID string) (int64, error) {
	return s.skuRepository.CountByProductID(ctx, strings.TrimSpace(productID))
}

func (s *Service) Create(ctx context.Context, input NewProduct) (*Product, error) {
	productID, err := s.idGenerator.New()
	if err != nil {
		return nil, fmt.Errorf("generate product id: %w", err)
	}

	now := s.clock.Now()
	item := Product{
		ID:                productID,
		Market:            input.Market,
		ExternalProductID: strings.TrimSpace(input.ExternalProductID),
		OriginalURL:       strings.TrimSpace(input.OriginalURL),
		Title:             strings.TrimSpace(input.Title),
		MainImageURL:      strings.TrimSpace(input.MainImageURL),
		CurrentPrice:      strings.TrimSpace(input.CurrentPrice),
		Currency:          strings.TrimSpace(input.Currency),
		ProductURL:        strings.TrimSpace(input.ProductURL),
		CollectionSource:  strings.TrimSpace(input.CollectionSource),
		LastCollectedAt:   now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.writer.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	for _, skuInput := range input.SKUs {
		skuID, err := s.idGenerator.New()
		if err != nil {
			return nil, fmt.Errorf("generate sku id: %w", err)
		}
		sku := SKU{
			ID:            skuID,
			ProductID:     item.ID,
			ExternalSKUID: strings.TrimSpace(skuInput.ExternalSKUID),
			OriginSKUID:   strings.TrimSpace(skuInput.OriginSKUID),
			SKUName:       strings.TrimSpace(skuInput.SKUName),
			Color:         strings.TrimSpace(skuInput.Color),
			Size:          strings.TrimSpace(skuInput.Size),
			Price:         strings.TrimSpace(skuInput.Price),
			OriginalPrice: strings.TrimSpace(skuInput.OriginalPrice),
			Currency:      strings.TrimSpace(skuInput.Currency),
			ImageURL:      strings.TrimSpace(skuInput.ImageURL),
			SKUProperties: strings.TrimSpace(skuInput.SKUProperties),
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := s.skuRepository.Create(ctx, sku); err != nil {
			return nil, fmt.Errorf("create sku: %w", err)
		}
	}

	return &item, nil
}

func (s *Service) CreateOrUpdateByMarketAndExternalProductID(ctx context.Context, input NewProduct) (*Product, error) {
	externalProductID := strings.TrimSpace(input.ExternalProductID)

	found, err := s.finder.FindByMarketAndExternalProductID(ctx, input.Market, externalProductID)
	if err != nil {
		return nil, fmt.Errorf("find product: %w", err)
	}
	if found == nil {
		return s.Create(ctx, input)
	}

	now := s.clock.Now()
	found.OriginalURL = strings.TrimSpace(input.OriginalURL)
	found.Title = strings.TrimSpace(input.Title)
	found.MainImageURL = strings.TrimSpace(input.MainImageURL)
	found.CurrentPrice = strings.TrimSpace(input.CurrentPrice)
	found.Currency = strings.TrimSpace(input.Currency)
	found.ProductURL = strings.TrimSpace(input.ProductURL)
	found.CollectionSource = strings.TrimSpace(input.CollectionSource)
	found.LastCollectedAt = now
	found.UpdatedAt = now

	if err := s.writer.Update(ctx, *found); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	return found, nil
}

func (s *Service) RefreshCollectedMetadata(ctx context.Context, productID string, title, mainImageURL, productURL string, collectedAt time.Time) (*Product, bool, error) {
	found, err := s.finder.FindByID(ctx, strings.TrimSpace(productID))
	if err != nil {
		return nil, false, fmt.Errorf("find product: %w", err)
	}
	if found == nil {
		return nil, false, coreerror.New(coreerror.ResourceNotFound)
	}

	changed := false

	title = strings.TrimSpace(title)
	mainImageURL = strings.TrimSpace(mainImageURL)
	productURL = strings.TrimSpace(productURL)

	if title != "" && found.Title != title {
		found.Title = title
		changed = true
	}
	if mainImageURL != "" && found.MainImageURL != mainImageURL {
		found.MainImageURL = mainImageURL
		changed = true
	}
	if productURL != "" && found.ProductURL != productURL {
		found.ProductURL = productURL
		changed = true
	}

	if collectedAt.IsZero() {
		collectedAt = s.clock.Now()
	}
	found.LastCollectedAt = collectedAt
	found.UpdatedAt = collectedAt

	if err := s.writer.Update(ctx, *found); err != nil {
		return nil, false, fmt.Errorf("update product: %w", err)
	}

	return found, changed, nil
}

func (s *Service) EnrichSKUs(ctx context.Context, productID string, skuInputs []NewSKU) (int, error) {
	found, err := s.finder.FindByID(ctx, strings.TrimSpace(productID))
	if err != nil {
		return 0, fmt.Errorf("find product: %w", err)
	}
	if found == nil {
		return 0, coreerror.New(coreerror.ResourceNotFound)
	}

	now := s.clock.Now()
	upserted := 0

	for _, skuInput := range skuInputs {
		externalSKUID := strings.TrimSpace(skuInput.ExternalSKUID)

		skuID, err := s.idGenerator.New()
		if err != nil {
			return upserted, fmt.Errorf("generate sku id: %w", err)
		}

		sku := SKU{
			ID:            skuID,
			ProductID:     productID,
			ExternalSKUID: externalSKUID,
			OriginSKUID:   strings.TrimSpace(skuInput.OriginSKUID),
			SKUName:       strings.TrimSpace(skuInput.SKUName),
			Color:         strings.TrimSpace(skuInput.Color),
			Size:          strings.TrimSpace(skuInput.Size),
			Price:         strings.TrimSpace(skuInput.Price),
			OriginalPrice: strings.TrimSpace(skuInput.OriginalPrice),
			Currency:      strings.TrimSpace(skuInput.Currency),
			ImageURL:      strings.TrimSpace(skuInput.ImageURL),
			SKUProperties: strings.TrimSpace(skuInput.SKUProperties),
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := s.skuRepository.Upsert(ctx, sku); err != nil {
			return upserted, fmt.Errorf("upsert sku: %w", err)
		}
		upserted++
	}

	return upserted, nil
}
