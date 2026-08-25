package testastic

import "fmt"

func matcherPlaceholder(prefix string, start int, literalValues map[string]struct{}) (string, int) {
	for index := start; ; index++ {
		placeholder := fmt.Sprintf("%s%d__", prefix, index)
		if _, exists := literalValues[placeholder]; !exists {
			return placeholder, index + 1
		}
	}
}

func stringValues(data any) map[string]struct{} {
	values := make(map[string]struct{})
	collectStringValues(data, values)

	return values
}

func collectStringValues(data any, values map[string]struct{}) {
	switch value := data.(type) {
	case map[string]any:
		for _, child := range value {
			collectStringValues(child, values)
		}
	case []any:
		for _, child := range value {
			collectStringValues(child, values)
		}
	case string:
		values[value] = struct{}{}
	}
}
