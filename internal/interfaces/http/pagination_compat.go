package http

import (
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/rs/zerolog/log"
)

const (
	legacyPaginationQueryParam   = "legacy"
	legacyPaginationHeader       = "X-Pagination-Format"
	legacyPaginationToggleHeader = "X-Legacy-Pagination"
	legacyPaginationDeprecation  = "2026-07-31"
)

func respondPaginated(c *fiber.Ctx, endpoint string, payload any) error {
	items, total, limit, offset, ok := extractPaginationFields(payload)
	if !ok {
		return c.JSON(payload)
	}
	return respondPaginatedValues(c, endpoint, items, total, limit, offset)
}

func respondPaginatedValues(c *fiber.Ctx, endpoint string, items any, total any, limit, offset int) error {
	if useLegacyPagination(c, endpoint) {
		c.Set(legacyPaginationHeader, "legacy")
		logLegacyPaginationUsage(c, endpoint)
		return c.JSON(fiber.Map{
			"items": items,
			"page": dto.PageResponse{
				Total:  asInt(total),
				Limit:  limit,
				Offset: offset,
			},
		})
	}

	c.Set(legacyPaginationHeader, "v2")
	return c.JSON(fiber.Map{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func useLegacyPagination(c *fiber.Ctx, endpoint string) bool {
	if !parseBoolEnv("PAGINATION_LEGACY_COMPAT_ENABLED", true) {
		return false
	}
	if !isLegacyEndpointEnabled(endpoint) {
		return false
	}
	if parseTruthy(c.Query(legacyPaginationQueryParam)) {
		return true
	}
	formatHeader := strings.TrimSpace(strings.ToLower(c.Get(legacyPaginationHeader)))
	if formatHeader == "legacy" || formatHeader == "v1" {
		return true
	}
	return parseTruthy(c.Get(legacyPaginationToggleHeader))
}

func isLegacyEndpointEnabled(endpoint string) bool {
	allowlist := strings.TrimSpace(os.Getenv("PAGINATION_LEGACY_COMPAT_ENDPOINTS"))
	if allowlist == "" || allowlist == "*" {
		return true
	}
	for _, raw := range strings.Split(allowlist, ",") {
		candidate := strings.TrimSpace(raw)
		if candidate == "" {
			continue
		}
		if candidate == endpoint {
			return true
		}
	}
	return false
}

func logLegacyPaginationUsage(c *fiber.Ctx, endpoint string) {
	log.Info().
		Str("event", "pagination_legacy_response_served").
		Str("endpoint", endpoint).
		Str("method", c.Method()).
		Str("path", c.Path()).
		Str("company_id", GetCompanyID(c)).
		Str("deprecation_date", legacyPaginationDeprecation).
		Msg("legacy pagination compatibility mode")
}

func extractPaginationFields(payload any) (items any, total any, limit int, offset int, ok bool) {
	v := reflect.ValueOf(payload)
	if !v.IsValid() {
		return nil, nil, 0, 0, false
	}
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil, 0, 0, false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, nil, 0, 0, false
	}

	itemsField := v.FieldByName("Items")
	totalField := v.FieldByName("Total")
	limitField := v.FieldByName("Limit")
	offsetField := v.FieldByName("Offset")
	if !itemsField.IsValid() || !totalField.IsValid() || !limitField.IsValid() || !offsetField.IsValid() {
		return nil, nil, 0, 0, false
	}

	limit, okLimit := reflectIntValue(limitField)
	offset, okOffset := reflectIntValue(offsetField)
	if !okLimit || !okOffset {
		return nil, nil, 0, 0, false
	}

	return itemsField.Interface(), totalField.Interface(), limit, offset, true
}

func reflectIntValue(v reflect.Value) (int, bool) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(v.Uint()), true
	default:
		return 0, false
	}
}

func asInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case float32:
		return int(n)
	case float64:
		return int(n)
	case string:
		out, _ := strconv.Atoi(strings.TrimSpace(n))
		return out
	default:
		return 0
	}
}

func parseBoolEnv(key string, defaultValue bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return defaultValue
	}
	return parseTruthy(v)
}

func parseTruthy(v string) bool {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "1", "true", "t", "yes", "y", "on", "legacy", "v1":
		return true
	default:
		return false
	}
}
