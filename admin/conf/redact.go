package conf

import (
	"fmt"
	"strings"
)

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

var commandSecretKeywords = []string{
	"token",
	"secret",
	"password",
	"passwd",
	"session_key",
	"db_conf",
	"dsn",
	"api_key",
	"access_key",
	"access_token",
	"refresh_token",
	"client_secret",
	"app_secret",
}

func MaskStoredSecret(fieldName, value string) string {
	if value == "" {
		return ""
	}

	return fmt.Sprintf("[redacted %s]", fieldName)
}

func MergeMaskedStoredSecret(fieldName, incomingValue, storedValue string) string {
	if storedValue == "" {
		return incomingValue
	}

	if incomingValue == MaskStoredSecret(fieldName, storedValue) {
		return storedValue
	}

	return incomingValue
}

func MaskCommandSecrets(command string) string {
	lines := strings.Split(command, "\n")
	for i, line := range lines {
		prefix, key, value, ok := parseCommandLine(line)
		if !ok || !isSensitiveCommandFlag(key) {
			continue
		}
		lines[i] = prefix + maskCommandValue(key, value)
	}

	return strings.Join(lines, "\n")
}

func MergeMaskedCommand(incomingCommand, storedCommand string) string {
	if storedCommand == "" || incomingCommand == "" {
		return incomingCommand
	}

	storedValues := make(map[string]string)
	for _, line := range strings.Split(storedCommand, "\n") {
		_, key, value, ok := parseCommandLine(line)
		if !ok {
			continue
		}
		storedValues[strings.ToLower(key)] = value
	}

	lines := strings.Split(incomingCommand, "\n")
	for i, line := range lines {
		prefix, key, value, ok := parseCommandLine(line)
		if !ok || !isSensitiveCommandFlag(key) {
			continue
		}

		rawValue, ok := storedValues[strings.ToLower(key)]
		if !ok {
			continue
		}

		if value == maskCommandValue(key, rawValue) {
			lines[i] = prefix + rawValue
		}
	}

	return strings.Join(lines, "\n")
}

func parseCommandLine(line string) (prefix, key, value string, ok bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, "-") {
		return "", "", "", false
	}

	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return "", "", "", false
	}

	indent := line[:len(line)-len(trimmed)]
	key = strings.TrimLeft(parts[0], "-")
	if key == "" {
		return "", "", "", false
	}

	return indent + parts[0] + "=", key, parts[1], true
}

func isSensitiveCommandFlag(key string) bool {
	lowerKey := strings.ToLower(strings.TrimSpace(key))
	if lowerKey == "" {
		return false
	}

	if strings.HasSuffix(lowerKey, "_file") {
		return false
	}

	for _, keyword := range commandSecretKeywords {
		if strings.Contains(lowerKey, keyword) {
			return true
		}
	}

	return false
}

func maskCommandValue(key, value string) string {
	lowerKey := strings.ToLower(strings.TrimSpace(key))
	if strings.Contains(lowerKey, "db_conf") || strings.Contains(lowerKey, "dsn") {
		return maskMaybeDSN(value)
	}

	return maskSecret(value)
}
