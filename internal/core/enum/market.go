package enum

type Market string

const (
	MarketAliExpress Market = "ALIEXPRESS"
	MarketCoupang    Market = "COUPANG"
)

func (m Market) IsSupported() bool {
	switch m {
	case MarketAliExpress, MarketCoupang:
		return true
	default:
		return false
	}
}
