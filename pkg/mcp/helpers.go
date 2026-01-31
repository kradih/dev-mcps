package mcp

import (
	"fmt"
)

func GetStringParam(params map[string]interface{}, key string, required bool) (string, error) {
	v, ok := params[key]
	if !ok {
		if required {
			return "", fmt.Errorf("missing required parameter: %s", key)
		}
		return "", nil
	}

	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s must be a string", key)
	}
	return s, nil
}

func GetIntParam(params map[string]interface{}, key string, required bool, defaultValue int) (int, error) {
	v, ok := params[key]
	if !ok {
		if required {
			return 0, fmt.Errorf("missing required parameter: %s", key)
		}
		return defaultValue, nil
	}

	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case int64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("parameter %s must be an integer", key)
	}
}

func GetBoolParam(params map[string]interface{}, key string, defaultValue bool) (bool, error) {
	v, ok := params[key]
	if !ok {
		return defaultValue, nil
	}

	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("parameter %s must be a boolean", key)
	}
	return b, nil
}

func GetStringArrayParam(params map[string]interface{}, key string, required bool) ([]string, error) {
	v, ok := params[key]
	if !ok {
		if required {
			return nil, fmt.Errorf("missing required parameter: %s", key)
		}
		return nil, nil
	}

	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("parameter %s must be an array", key)
	}

	result := make([]string, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("parameter %s[%d] must be a string", key, i)
		}
		result[i] = s
	}
	return result, nil
}

func GetMapParam(params map[string]interface{}, key string, required bool) (map[string]string, error) {
	v, ok := params[key]
	if !ok {
		if required {
			return nil, fmt.Errorf("missing required parameter: %s", key)
		}
		return nil, nil
	}

	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parameter %s must be an object", key)
	}

	result := make(map[string]string)
	for k, val := range m {
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("parameter %s.%s must be a string", key, k)
		}
		result[k] = s
	}
	return result, nil
}
