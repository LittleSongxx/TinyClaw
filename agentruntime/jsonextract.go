package agentruntime

import "fmt"

func ExtractJSONObject(input string) (string, error) {
	start := -1
	depth := 0
	inString := false
	escaped := false

	for i, r := range input {
		switch {
		case escaped:
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"':
			inString = !inString
		case inString:
			continue
		case r == '{':
			if depth == 0 {
				start = i
			}
			depth++
		case r == '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				return input[start : i+1], nil
			}
		}
	}

	return "", fmt.Errorf("json object not found")
}
