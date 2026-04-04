package enum

import "strings"

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

func CurrencyForLanguage(language string) string {
	switch strings.ToUpper(strings.TrimSpace(language)) {
	case "", "KO":
		return "KRW"
	case "EN":
		return "USD"
	default:
		return "KRW"
	}
}

func NormalizeLanguage(language string) string {
	language = strings.ToUpper(strings.TrimSpace(language))
	if language == "" {
		return "KO"
	}
	switch language {
	case "KO", "EN":
		return language
	default:
		return "KO"
	}
}
