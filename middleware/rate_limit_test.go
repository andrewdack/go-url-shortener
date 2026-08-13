package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/andrewdack/go-url-shortener/store"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	service, err := store.NewStore(context.Background())
	if err != nil {
		panic(err)
	}
	testStoreService = service
	code := m.Run()
	_ = service.Close()
	os.Exit(code)
}

var testStoreService *store.StorageService

func TestRateLimitAllowsRequestsUpToLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	clientIP := uniqueTestIP("198")
	window := time.Minute

	router := gin.New()
	router.Use(RateLimit(testStoreService, 2, window))
	router.POST("/resource", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for requestNumber := 1; requestNumber <= 2; requestNumber++ {
		request := httptest.NewRequest(http.MethodPost, "/resource", nil)
		request.RemoteAddr = clientIP + ":1234"
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assert.Equal(t, http.StatusNoContent, response.Code, "request %d should be allowed", requestNumber)
	}
}

func TestRateLimitRejectsRequestsOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	clientIP := uniqueTestIP("203")

	router := gin.New()
	router.Use(RateLimit(testStoreService, 1, time.Minute))
	router.POST("/resource", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	firstRequest := httptest.NewRequest(http.MethodPost, "/resource", nil)
	firstRequest.RemoteAddr = clientIP + ":1234"
	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, firstRequest)
	require.Equal(t, http.StatusNoContent, firstResponse.Code)

	secondRequest := httptest.NewRequest(http.MethodPost, "/resource", nil)
	secondRequest.RemoteAddr = clientIP + ":1234"
	secondResponse := httptest.NewRecorder()
	router.ServeHTTP(secondResponse, secondRequest)

	assert.Equal(t, http.StatusTooManyRequests, secondResponse.Code)
	assert.JSONEq(t, `{"error":"rate limit exceeded, try again later"}`, secondResponse.Body.String())
}

func uniqueTestIP(prefix string) string {
	value := uint64(time.Now().UnixNano())
	return fmt.Sprintf("%s.%d.%d.%d", prefix, byte(value>>16), byte(value>>8), byte(value))
}
