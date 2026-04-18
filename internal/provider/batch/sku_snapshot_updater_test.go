package batch

import (
	"context"
	"reflect"
	"testing"
	"time"

	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
	"github.com/ljj/gugu-admin-api/internal/core/enum"
)

type fakeProductFinder struct {
	collectionSourceProducts map[string][]domainproduct.Product
	trackedProductIDs        []string
	productsByID             map[string]domainproduct.Product
	candidateProducts        []domainproduct.Product

	lastCollectionSource string
	lastFindByIDs        []string
	lastCandidateFilter  domainproduct.PriceUpdateCandidateFilter
}

func (f *fakeProductFinder) FindByID(ctx context.Context, productID string) (*domainproduct.Product, error) {
	if f.productsByID == nil {
		return nil, nil
	}
	p, ok := f.productsByID[productID]
	if !ok {
		return nil, nil
	}
	return &p, nil
}

func (f *fakeProductFinder) FindByIDs(ctx context.Context, productIDs []string) ([]domainproduct.Product, error) {
	f.lastFindByIDs = append([]string{}, productIDs...)
	if f.productsByID == nil {
		return nil, nil
	}

	result := make([]domainproduct.Product, 0, len(productIDs))
	for _, productID := range productIDs {
		if p, ok := f.productsByID[productID]; ok {
			result = append(result, p)
		}
	}
	return result, nil
}

func (f *fakeProductFinder) FindByMarketAndExternalProductID(ctx context.Context, market enum.Market, externalProductID string) (*domainproduct.Product, error) {
	return nil, nil
}

func (f *fakeProductFinder) ListActiveTrackedProductIDs(ctx context.Context) ([]string, error) {
	return append([]string{}, f.trackedProductIDs...), nil
}

func (f *fakeProductFinder) ListByMarket(ctx context.Context, market enum.Market) ([]domainproduct.Product, error) {
	return nil, nil
}

func (f *fakeProductFinder) ListByCollectionSource(ctx context.Context, collectionSource string) ([]domainproduct.Product, error) {
	f.lastCollectionSource = collectionSource
	if f.collectionSourceProducts == nil {
		return nil, nil
	}
	return append([]domainproduct.Product{}, f.collectionSourceProducts[collectionSource]...), nil
}

func (f *fakeProductFinder) ListAllLocalized(ctx context.Context, language string) ([]domainproduct.LocalizedProduct, error) {
	return nil, nil
}

func (f *fakeProductFinder) ListByCollectionSourceLocalized(ctx context.Context, collectionSource, language string) ([]domainproduct.LocalizedProduct, error) {
	return nil, nil
}

func (f *fakeProductFinder) ListPriceUpdateCandidates(ctx context.Context, filter domainproduct.PriceUpdateCandidateFilter) ([]domainproduct.Product, error) {
	f.lastCandidateFilter = filter
	return append([]domainproduct.Product{}, f.candidateProducts...), nil
}

func (f *fakeProductFinder) ListAll(ctx context.Context) ([]domainproduct.Product, error) {
	return nil, nil
}

type noopProductWriter struct{}

func (noopProductWriter) Create(ctx context.Context, product domainproduct.Product) error {
	return nil
}

func (noopProductWriter) Update(ctx context.Context, product domainproduct.Product) error {
	return nil
}

type noopSKURepository struct{}

func (noopSKURepository) Create(ctx context.Context, sku domainproduct.SKU) error {
	return nil
}

func (noopSKURepository) Upsert(ctx context.Context, sku domainproduct.SKU) error {
	return nil
}

func (noopSKURepository) FindByID(ctx context.Context, skuID string) (*domainproduct.SKU, error) {
	return nil, nil
}

func (noopSKURepository) FindByProductID(ctx context.Context, productID string) ([]domainproduct.SKU, error) {
	return nil, nil
}

func (noopSKURepository) FindByProductIDAndExternalSKUID(ctx context.Context, productID string, externalSKUID string) (*domainproduct.SKU, error) {
	return nil, nil
}

func (noopSKURepository) CountByProductID(ctx context.Context, productID string) (int64, error) {
	return 0, nil
}

type noopIDGenerator struct{}

func (noopIDGenerator) New() (string, error) {
	return "test-id", nil
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC)
}

func newTestProductService(finder domainproduct.Finder) *domainproduct.Service {
	return domainproduct.NewService(
		finder,
		noopProductWriter{},
		noopSKURepository{},
		noopIDGenerator{},
		fixedClock{},
	)
}

func TestResolveTargets_TargetGroupHotProducts(t *testing.T) {
	hotProducts := []domainproduct.Product{
		{ID: "hot-1", CollectionSource: domainproduct.CollectionSourceHotProductQuery},
		{ID: "hot-2", CollectionSource: domainproduct.CollectionSourceHotProductQuery},
	}

	finder := &fakeProductFinder{
		collectionSourceProducts: map[string][]domainproduct.Product{
			domainproduct.CollectionSourceHotProductQuery: hotProducts,
		},
	}

	updater := &SKUSnapshotUpdater{productService: newTestProductService(finder)}

	got, err := updater.resolveTargets(context.Background(), PriceUpdateFilter{
		TargetGroup: TargetGroupHotProducts,
	})
	if err != nil {
		t.Fatalf("resolveTargets() error = %v", err)
	}

	if finder.lastCollectionSource != domainproduct.CollectionSourceHotProductQuery {
		t.Fatalf("ListByCollectionSource called with %q, want %q", finder.lastCollectionSource, domainproduct.CollectionSourceHotProductQuery)
	}
	if !reflect.DeepEqual(got, hotProducts) {
		t.Fatalf("resolveTargets() = %+v, want %+v", got, hotProducts)
	}
}

func TestResolveTargets_TargetGroupTracked(t *testing.T) {
	finder := &fakeProductFinder{
		trackedProductIDs: []string{"p1", "p2"},
		productsByID: map[string]domainproduct.Product{
			"p1": {ID: "p1"},
			"p2": {ID: "p2"},
		},
	}

	updater := &SKUSnapshotUpdater{productService: newTestProductService(finder)}

	got, err := updater.resolveTargets(context.Background(), PriceUpdateFilter{
		TargetGroup: TargetGroupTracked,
	})
	if err != nil {
		t.Fatalf("resolveTargets() error = %v", err)
	}

	if !reflect.DeepEqual(finder.lastFindByIDs, []string{"p1", "p2"}) {
		t.Fatalf("FindByIDs called with %+v, want %+v", finder.lastFindByIDs, []string{"p1", "p2"})
	}
	if len(got) != 2 {
		t.Fatalf("resolveTargets() count = %d, want 2", len(got))
	}
}

func TestResolveTargets_TargetGroupAll(t *testing.T) {
	candidates := []domainproduct.Product{
		{ID: "all-1", Market: enum.MarketAliExpress, CollectionSource: "OTHER"},
	}

	finder := &fakeProductFinder{
		candidateProducts: candidates,
	}

	updater := &SKUSnapshotUpdater{productService: newTestProductService(finder)}

	filter := PriceUpdateFilter{
		CollectionSource: "OTHER",
		Market:           enum.MarketAliExpress,
	}

	got, err := updater.resolveTargets(context.Background(), filter)
	if err != nil {
		t.Fatalf("resolveTargets() error = %v", err)
	}

	if finder.lastCandidateFilter.CollectionSource != filter.CollectionSource {
		t.Fatalf("ListPriceUpdateCandidates collection_source = %q, want %q", finder.lastCandidateFilter.CollectionSource, filter.CollectionSource)
	}
	if finder.lastCandidateFilter.Market != filter.Market {
		t.Fatalf("ListPriceUpdateCandidates market = %q, want %q", finder.lastCandidateFilter.Market, filter.Market)
	}
	if !reflect.DeepEqual(got, candidates) {
		t.Fatalf("resolveTargets() = %+v, want %+v", got, candidates)
	}
}
