package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClickCounterIncrementsOnlySuccessfulRedirects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shortURL := fmt.Sprintf("middleware-click-%d", time.Now().UnixNano())
	ownerID := "middleware-test-owner"
	require.NoError(t, testStoreService.SaveUrlMapping(context.Background(), shortURL, "https://example.com/click-counter", ownerID))
	t.Cleanup(func() {
		_, _ = testStoreService.DeleteUrlMapping(context.Background(), shortURL, ownerID)
	})

	successRouter := gin.New()
	successRouter.GET("/:shortUrl", ClickCounter(testStoreService), func(c *gin.Context) {
		c.Redirect(http.StatusFound, "https://example.com/click-counter")
	})
	successRequest := httptest.NewRequest(http.MethodGet, "/"+shortURL, nil)
	successResponse := httptest.NewRecorder()
	successRouter.ServeHTTP(successResponse, successRequest)

	require.Equal(t, http.StatusFound, successResponse.Code)
	clicks, err := testStoreService.RetrieveClickCount(context.Background(), shortURL)
	require.NoError(t, err)
	assert.Equal(t, int64(1), clicks)

	failureRouter := gin.New()
	failureRouter.GET("/:shortUrl", ClickCounter(testStoreService), func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})
	failureRequest := httptest.NewRequest(http.MethodGet, "/"+shortURL, nil)
	failureResponse := httptest.NewRecorder()
	failureRouter.ServeHTTP(failureResponse, failureRequest)

	assert.Equal(t, http.StatusNotFound, failureResponse.Code)
	clicks, err = testStoreService.RetrieveClickCount(context.Background(), shortURL)
	require.NoError(t, err)
	assert.Equal(t, int64(1), clicks, "failed responses must not increment clicks")
}
