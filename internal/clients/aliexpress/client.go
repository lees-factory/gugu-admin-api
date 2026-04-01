package aliexpress

import "context"

type AffiliateProductDetailRequest struct {
	ProductIDs     []string
	TargetCurrency string
	TargetLanguage string
	Country        string
}

type HotProductQueryRequest struct {
	CategoryIDs    []string
	Keywords       string
	PageNo         int
	PageSize       int
	Sort           string
	MinSalePrice   string
	MaxSalePrice   string
	ShipToCountry  string
	TargetCurrency string
	TargetLanguage string
}

type HotProduct struct {
	ProductID               string
	ProductTitle            string
	ProductMainImageURL     string
	ProductDetailURL        string
	SalePrice               string
	SalePriceCurrency       string
	TargetSalePrice         string
	TargetSalePriceCurrency string
}

type AffiliateProductDetail struct {
	ProductID               string
	ProductTitle            string
	ProductDetailURL        string
	ProductMainImageURL     string
	SalePrice               string
	SalePriceCurrency       string
	TargetSalePrice         string
	TargetSalePriceCurrency string
}

type DropshippingProductRequest struct {
	ProductID             string
	ShipToCountry         string
	TargetCurrency        string
	TargetLanguage        string
	RemovePersonalBenefit bool
}

type DropshippingProductDetail struct {
	ProductID    string
	Subject      string
	CurrencyCode string
	ImageURLs    []string
	SKUs         []DropshippingSKU
}

type DropshippingSKU struct {
	SKUID          string
	OriginSKUID    string
	SKUAttr        string
	Price          string
	OfferSalePrice string
	CurrencyCode   string
	ImageURL       string
	SKUName        string
	Color          string
	Size           string
}

type Client interface {
	QueryHotProducts(ctx context.Context, req HotProductQueryRequest) ([]HotProduct, error)
	GetAffiliateProductDetails(ctx context.Context, req AffiliateProductDetailRequest) ([]AffiliateProductDetail, error)
	GetDropshippingProduct(ctx context.Context, req DropshippingProductRequest) (*DropshippingProductDetail, error)
}
