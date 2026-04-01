package product

import "time"

type SKU struct {
	ID            string
	ProductID     string
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
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
