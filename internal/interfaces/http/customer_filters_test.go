package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCustomerListFilters_PrioritizesCategoryID(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		filters := parseCustomerListFilters(c)
		assert.Equal(t, "dbc1cd15-bd29-431b-ae9d-d9132ed72745", filters.CategoryID)
		assert.Equal(t, "", filters.CategoryName)
		assert.False(t, filters.WithoutCategory)
		return c.SendStatus(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/?category_id=dbc1cd15-bd29-431b-ae9d-d9132ed72745&filter=category:VIP", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestParseCustomerListFilters_ReadsCategoryIDFromFilter(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		filters := parseCustomerListFilters(c)
		assert.Equal(t, "dbc1cd15-bd29-431b-ae9d-d9132ed72745", filters.CategoryID)
		assert.Equal(t, "", filters.CategoryName)
		return c.SendStatus(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/?filter=category_id:dbc1cd15-bd29-431b-ae9d-d9132ed72745", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestParseCustomerListFilters_SupportsLegacyCategoryByName(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		filters := parseCustomerListFilters(c)
		assert.Equal(t, "", filters.CategoryID)
		assert.Equal(t, "VIP", filters.CategoryName)
		return c.SendStatus(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/?filter=category:VIP", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestParseCustomerListFilters_WithoutCategoryOverridesOthers(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		filters := parseCustomerListFilters(c)
		assert.True(t, filters.WithoutCategory)
		assert.Equal(t, "", filters.CategoryID)
		assert.Equal(t, "", filters.CategoryIDFallback)
		assert.Equal(t, "", filters.CategoryName)
		return c.SendStatus(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/?without_category=true&category_id=dbc1cd15-bd29-431b-ae9d-d9132ed72745&filter=category:VIP", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}
