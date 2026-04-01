package product

import "github.com/ljj/gugu-admin-api/internal/core/enum"

type NewProduct struct {
	Market            enum.Market
	ExternalProductID string
	OriginalURL       string
	Title             string
	MainImageURL      string
	CurrentPrice      string
	Currency          string
	ProductURL        string
	CollectionSource  string
	SKUs              []NewSKU
}

type NewSKU struct {
	ExternalSKUID string
	OriginSKUID   string
	SKUName       string
	Color         string
	Size          string
	Price         string
	OriginalPrice string
	Currency      string
	ImageURL      string
	SKUProperties string
}
