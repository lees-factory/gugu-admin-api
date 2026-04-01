package enum

var SupportedCurrencies = []string{"KRW", "USD"}

func LanguageForCurrency(currency string) string {
	switch currency {
	case "KRW":
		return "KO"
	case "USD":
		return "EN"
	default:
		return "EN"
	}
}
