package main

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

var strictMode bool

// strictValidateSpec rejects specs with validation warnings in strict mode
func strictValidateSpec(doc *openapi3.T) error {
	err := doc.Validate(context.Background())
	if err != nil {
		if strictMode {
			return fmt.Errorf("strict mode: spec validation failed: %w", err)
		}
		logStrictWarning("spec validation", err.Error())
	}
	return nil
}

// validateResponseAgainstSchema validates generated response data against schema constraints
// returns a list of warnings (does not block response in non-strict mode)
func validateResponseAgainstSchema(schema *openapi3.SchemaRef, data interface{}, path string) []string {
	if schema == nil || schema.Value == nil || data == nil {
		return nil
	}
	s := schema.Value
	var warnings []string

	types := s.Type.Slice()
	if len(types) == 0 && len(s.Properties) > 0 {
		types = []string{"object"}
	}
	if len(types) == 0 {
		return nil
	}

	switch types[0] {
	case "object":
		m, ok := data.(map[string]interface{})
		if !ok {
			return []string{fmt.Sprintf("%s: expected object, got %T", path, data)}
		}
		// check required fields
		for _, req := range s.Required {
			if _, exists := m[req]; !exists {
				warnings = append(warnings, fmt.Sprintf("%s: missing required field '%s'", path, req))
			}
		}
		// validate each property
		for name, prop := range s.Properties {
			if val, exists := m[name]; exists {
				sub := validateResponseAgainstSchema(prop, val, path+"."+name)
				warnings = append(warnings, sub...)
			}
		}

	case "array":
		arr, ok := data.([]interface{})
		if !ok {
			return []string{fmt.Sprintf("%s: expected array, got %T", path, data)}
		}
		if s.MinItems > 0 && uint64(len(arr)) < s.MinItems {
			warnings = append(warnings, fmt.Sprintf("%s: array has %d items, minimum is %d", path, len(arr), s.MinItems))
		}
		if s.MaxItems != nil && uint64(len(arr)) > *s.MaxItems {
			warnings = append(warnings, fmt.Sprintf("%s: array has %d items, maximum is %d", path, len(arr), *s.MaxItems))
		}
		if s.Items != nil {
			for i, item := range arr {
				sub := validateResponseAgainstSchema(s.Items, item, fmt.Sprintf("%s[%d]", path, i))
				warnings = append(warnings, sub...)
			}
		}

	case "string":
		str, ok := data.(string)
		if !ok {
			return []string{fmt.Sprintf("%s: expected string, got %T", path, data)}
		}
		if s.MinLength > 0 && uint64(len(str)) < s.MinLength {
			warnings = append(warnings, fmt.Sprintf("%s: string length %d is below minLength %d", path, len(str), s.MinLength))
		}
		if s.MaxLength != nil && uint64(len(str)) > *s.MaxLength {
			warnings = append(warnings, fmt.Sprintf("%s: string length %d exceeds maxLength %d", path, len(str), *s.MaxLength))
		}
		if s.Pattern != "" {
			re, err := regexp.Compile(s.Pattern)
			if err == nil && !re.MatchString(str) {
				warnings = append(warnings, fmt.Sprintf("%s: string '%s' doesn't match pattern '%s'", path, truncate(str, 30), s.Pattern))
			}
		}
		if len(s.Enum) > 0 {
			found := false
			for _, e := range s.Enum {
				if fmt.Sprintf("%v", e) == str {
					found = true
					break
				}
			}
			if !found {
				warnings = append(warnings, fmt.Sprintf("%s: value '%s' not in enum %v", path, truncate(str, 30), s.Enum))
			}
		}

	case "integer", "number":
		num, ok := toFloat64(data)
		if !ok {
			return []string{fmt.Sprintf("%s: expected number, got %T", path, data)}
		}
		if s.Min != nil && num < *s.Min {
			warnings = append(warnings, fmt.Sprintf("%s: value %v is below minimum %v", path, num, *s.Min))
		}
		if s.Max != nil && num > *s.Max {
			warnings = append(warnings, fmt.Sprintf("%s: value %v exceeds maximum %v", path, num, *s.Max))
		}
		// ExclusiveMin/ExclusiveMax are bools in this version of kin-openapi
		if s.ExclusiveMin && s.Min != nil && num <= *s.Min {
			warnings = append(warnings, fmt.Sprintf("%s: value %v must be > %v (exclusiveMinimum)", path, num, *s.Min))
		}
		if s.ExclusiveMax && s.Max != nil && num >= *s.Max {
			warnings = append(warnings, fmt.Sprintf("%s: value %v must be < %v (exclusiveMaximum)", path, num, *s.Max))
		}
		if s.MultipleOf != nil && *s.MultipleOf != 0 {
			if math.Mod(num, *s.MultipleOf) != 0 {
				warnings = append(warnings, fmt.Sprintf("%s: value %v is not a multiple of %v", path, num, *s.MultipleOf))
			}
		}
	}

	return warnings
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	}
	return 0, false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// strictValidateRequest performs more aggressive required field checking in strict mode
func strictValidateRequestBody(schema *openapi3.SchemaRef, body map[string]interface{}) []string {
	if schema == nil || schema.Value == nil {
		return nil
	}
	s := schema.Value
	var errors []string

	// check all required fields recursively
	for _, req := range s.Required {
		val, exists := body[req]
		if !exists {
			errors = append(errors, fmt.Sprintf("required field '%s' is missing", req))
		} else if val == nil {
			errors = append(errors, fmt.Sprintf("required field '%s' is null", req))
		} else if str, ok := val.(string); ok && strings.TrimSpace(str) == "" {
			errors = append(errors, fmt.Sprintf("required field '%s' is empty", req))
		}
	}

	// recurse into nested objects
	for name, prop := range s.Properties {
		if prop.Value == nil {
			continue
		}
		propTypes := prop.Value.Type.Slice()
		if len(propTypes) > 0 && propTypes[0] == "object" {
			if nested, ok := body[name].(map[string]interface{}); ok {
				sub := strictValidateRequestBody(prop, nested)
				for _, e := range sub {
					errors = append(errors, name+"."+e)
				}
			}
		}
	}

	return errors
}
