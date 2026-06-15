package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func ExtractPromptText(payload map[string]any, pointer string) (string, error) {
	value, err := valueAtPointer(payload, pointer)
	if err != nil {
		return "", fmt.Errorf("invalid_payload: %w", err)
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", errors.New("invalid_payload: prompt text missing")
	}
	return text, nil
}

func valueAtPointer(value any, pointer string) (any, error) {
	if pointer == "" || pointer[0] != '/' {
		return nil, errors.New("pointer must start with slash")
	}
	current := value
	for _, raw := range strings.Split(pointer[1:], "/") {
		next, err := pointerStep(current, decodePointerSegment(raw))
		if err != nil {
			return nil, err
		}
		current = next
	}
	return current, nil
}

func decodePointerSegment(value string) string {
	return strings.NewReplacer("~1", "/", "~0", "~").Replace(value)
}

func pointerStep(current any, segment string) (any, error) {
	switch value := current.(type) {
	case map[string]any:
		next, ok := value[segment]
		if !ok {
			return nil, fmt.Errorf("missing segment %s", segment)
		}
		return next, nil
	case []any:
		index, err := strconv.Atoi(segment)
		if err != nil || index < 0 || index >= len(value) {
			return nil, fmt.Errorf("invalid index %s", segment)
		}
		return value[index], nil
	default:
		return nil, fmt.Errorf("segment %s has no child", segment)
	}
}
