package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRespondPaginatedValues_DefaultsToV2(t *testing.T) {
	t.Setenv("PAGINATION_LEGACY_COMPAT_ENABLED", "true")

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/list", func(c *fiber.Ctx) error {
		return respondPaginatedValues(c, "test.endpoint", []string{"a"}, 7, 20, 0)
	})

	req := httptest.NewRequest(http.MethodGet, "/list", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "v2", resp.Header.Get(legacyPaginationHeader))

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	_, hasPage := body["page"]
	assert.False(t, hasPage)
	assert.Equal(t, float64(7), body["total"])
	assert.Equal(t, float64(20), body["limit"])
	assert.Equal(t, float64(0), body["offset"])
}

func TestRespondPaginatedValues_LegacyViaQueryFlag(t *testing.T) {
	t.Setenv("PAGINATION_LEGACY_COMPAT_ENABLED", "true")

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/list", func(c *fiber.Ctx) error {
		return respondPaginatedValues(c, "test.endpoint", []string{"a"}, 7, 20, 0)
	})

	req := httptest.NewRequest(http.MethodGet, "/list?legacy=true", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "legacy", resp.Header.Get(legacyPaginationHeader))

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	page, ok := body["page"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(7), page["total"])
	assert.Equal(t, float64(20), page["limit"])
	assert.Equal(t, float64(0), page["offset"])
	_, hasTotal := body["total"]
	assert.False(t, hasTotal)
}

func TestRespondPaginatedValues_LegacyDisabledByFeatureFlag(t *testing.T) {
	t.Setenv("PAGINATION_LEGACY_COMPAT_ENABLED", "false")

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/list", func(c *fiber.Ctx) error {
		return respondPaginatedValues(c, "test.endpoint", []string{"a"}, 7, 20, 0)
	})

	req := httptest.NewRequest(http.MethodGet, "/list?legacy=true", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "v2", resp.Header.Get(legacyPaginationHeader))

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	_, hasPage := body["page"]
	assert.False(t, hasPage)
	assert.Equal(t, float64(7), body["total"])
}
