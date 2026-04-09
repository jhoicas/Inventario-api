package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
)

func parseCustomerListFilters(c *fiber.Ctx) dto.CustomerListFilters {
	filters := dto.CustomerListFilters{
		Search:             strings.TrimSpace(c.Query("search")),
		Filter:             strings.TrimSpace(c.Query("filter")),
		CategoryID:         strings.TrimSpace(c.Query("category_id")),
		CategoryIDFallback: strings.TrimSpace(c.Query("categoryId")),
		CategoryName:       strings.TrimSpace(c.Query("category_name")),
	}

	filters.WithoutCategory = parseCustomerTruthy(c.Query("without_category")) || parseCustomerTruthy(c.Query("withoutCategory"))
	if filters.WithoutCategory {
		filters.CategoryID = ""
		filters.CategoryIDFallback = ""
		filters.CategoryName = ""
		return filters
	}

	if filters.CategoryID == "" && filters.CategoryIDFallback == "" && filters.Filter != "" {
		if id := extractFilterCategoryID(filters.Filter); id != "" {
			filters.CategoryID = id
		} else if name := extractFilterCategoryName(filters.Filter); name != "" && filters.CategoryName == "" {
			filters.CategoryName = name
		}
	}

	return filters
}

func extractFilterCategoryID(filter string) string {
	clean := strings.TrimSpace(filter)
	prefixes := []string{"category_id:", "category_id="}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(clean), prefix) {
			candidate := strings.TrimSpace(clean[len(prefix):])
			if isUUID(candidate) {
				return candidate
			}
		}
	}
	return ""
}

func extractFilterCategoryName(filter string) string {
	clean := strings.TrimSpace(filter)
	lower := strings.ToLower(clean)
	for _, prefix := range []string{"category:", "category=", "category_name:", "category_name="} {
		if strings.HasPrefix(lower, prefix) {
			return strings.TrimSpace(clean[len(prefix):])
		}
	}
	return ""
}

func parseCustomerTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func isUUID(value string) bool {
	_, err := uuid.Parse(strings.TrimSpace(value))
	return err == nil
}
