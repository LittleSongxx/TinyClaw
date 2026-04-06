package conf

import "strings"

func maskSecret(value string) string {
	if value == "" {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= 4 {
		return strings.Repeat("*", len(runes))
	}
	if len(runes) <= 8 {
		return string(runes[:1]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1:])
	}

	return string(runes[:2]) + strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-2:])
}

func maskMaybeDSN(value string) string {
	if value == "" {
		return ""
	}

	if strings.Contains(value, "@") || strings.Contains(value, "://") {
		return maskSecret(value)
	}

	return value
}
