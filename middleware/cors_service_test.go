package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newCORSTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS())
	router.GET("/resource", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.POST("/resource", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	return router
}

func TestCORSAllowsConfiguredOrigins(t *testing.T) {
	t.Parallel()

	origins := []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"https://andrewdack.github.io",
	}

	for _, origin := range origins {
		origin := origin
		t.Run(origin, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, "/resource", nil)
			request.Header.Set("Origin", origin)
			response := httptest.NewRecorder()

			newCORSTestRouter().ServeHTTP(response, request)

			assert.Equal(t, http.StatusNoContent, response.Code)
			assert.Equal(t, origin, response.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

func TestCORSHandlesPreflight(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	response := httptest.NewRecorder()

	newCORSTestRouter().ServeHTTP(response, request)

	assert.Equal(t, http.StatusNoContent, response.Code)
	assert.Equal(t, "http://localhost:5173", response.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, response.Header().Get("Access-Control-Allow-Methods"), http.MethodPost)
	assert.Contains(t, response.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	assert.NotEmpty(t, response.Header().Get("Access-Control-Max-Age"))
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	request.Header.Set("Origin", "https://example.invalid")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	response := httptest.NewRecorder()

	newCORSTestRouter().ServeHTTP(response, request)

	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.Empty(t, response.Header().Get("Access-Control-Allow-Origin"))
}
