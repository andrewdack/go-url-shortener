package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/andrewdack/go-url-shortener/shortener"
	"github.com/andrewdack/go-url-shortener/store"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	store.InitializeStore()
	os.Exit(m.Run())
}

func newHandlerContext(method string, path string, body string) (*gin.Context, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = request
	return context, response
}

func TestCreateShortUrlRejectsInvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, response := newHandlerContext(http.MethodPost, "/create-short-url", `{"longUrl":"not-a-url","userId":"user"}`)

	CreateShortUrl(context)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "error")
}

func TestCreateShortUrlReturnsAndPersistsShortUrl(t *testing.T) {
	gin.SetMode(gin.TestMode)
	longURL := fmt.Sprintf("https://example.com/create-%d", time.Now().UnixNano())
	userID := "handler-create-owner"
	shortURL := shortener.GenerateShortLink(longURL, userID)
	t.Cleanup(func() {
		_, _ = store.DeleteUrlMapping(shortURL, userID)
	})

	context, response := newHandlerContext(http.MethodPost, "/create-short-url", fmt.Sprintf(`{"longUrl":%q,"userId":%q}`, longURL, userID))
	CreateShortUrl(context)

	require.Equal(t, http.StatusOK, response.Code)
	var body struct {
		ShortURL string `json:"shortUrl"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Contains(t, body.ShortURL, shortURL)

	retrievedURL, err := store.RetrieveInitialUrl(shortURL)
	require.NoError(t, err)
	assert.Equal(t, longURL, retrievedURL)
}

func TestUpdateShortUrlEnforcesOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shortURL := fmt.Sprintf("handler-update-%d", time.Now().UnixNano())
	originalURL := "https://example.com/original"
	updatedURL := "https://example.com/updated"
	ownerID := "handler-update-owner"
	require.NoError(t, store.SaveUrlMapping(shortURL, originalURL, ownerID))
	t.Cleanup(func() {
		_, _ = store.DeleteUrlMapping(shortURL, ownerID)
	})

	wrongContext, wrongResponse := newHandlerContext(http.MethodPatch, "/"+shortURL, fmt.Sprintf(`{"longUrl":%q,"userId":"wrong-owner"}`, updatedURL))
	wrongContext.Params = gin.Params{{Key: "shortUrl", Value: shortURL}}
	UpdateShortUrl(wrongContext)
	assert.Equal(t, http.StatusForbidden, wrongResponse.Code)

	correctContext, correctResponse := newHandlerContext(http.MethodPatch, "/"+shortURL, fmt.Sprintf(`{"longUrl":%q,"userId":%q}`, updatedURL, ownerID))
	correctContext.Params = gin.Params{{Key: "shortUrl", Value: shortURL}}
	UpdateShortUrl(correctContext)
	assert.Equal(t, http.StatusOK, correctResponse.Code)

	retrievedURL, err := store.RetrieveInitialUrl(shortURL)
	require.NoError(t, err)
	assert.Equal(t, updatedURL, retrievedURL)
}

func TestHandleShortUrlRedirectReturnsDestination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shortURL := fmt.Sprintf("handler-redirect-%d", time.Now().UnixNano())
	ownerID := "handler-redirect-owner"
	destination := "https://example.com/redirect-destination"
	require.NoError(t, store.SaveUrlMapping(shortURL, destination, ownerID))
	t.Cleanup(func() {
		_, _ = store.DeleteUrlMapping(shortURL, ownerID)
	})

	context, response := newHandlerContext(http.MethodGet, "/"+shortURL, "")
	context.Params = gin.Params{{Key: "shortUrl", Value: shortURL}}
	HandleShortUrlRedirect(context)

	assert.Equal(t, http.StatusFound, response.Code)
	assert.Equal(t, destination, response.Header().Get("Location"))
}

func TestHandleShortUrlRedirectReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shortURL := fmt.Sprintf("handler-missing-%d", time.Now().UnixNano())
	context, response := newHandlerContext(http.MethodGet, "/"+shortURL, "")
	context.Params = gin.Params{{Key: "shortUrl", Value: shortURL}}

	HandleShortUrlRedirect(context)

	assert.Equal(t, http.StatusNotFound, response.Code)
	assert.Contains(t, response.Body.String(), "short url not found")
}

func TestDeleteShortUrlEnforcesOwnerAndRemovesMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shortURL := fmt.Sprintf("handler-delete-%d", time.Now().UnixNano())
	ownerID := "handler-delete-owner"
	require.NoError(t, store.SaveUrlMapping(shortURL, "https://example.com/delete", ownerID))

	wrongContext, wrongResponse := newHandlerContext(http.MethodDelete, "/"+shortURL, `{"userId":"wrong-owner"}`)
	wrongContext.Params = gin.Params{{Key: "shortUrl", Value: shortURL}}
	DeleteShortUrl(wrongContext)
	assert.Equal(t, http.StatusForbidden, wrongResponse.Code)

	correctContext, correctResponse := newHandlerContext(http.MethodDelete, "/"+shortURL, fmt.Sprintf(`{"userId":%q}`, ownerID))
	correctContext.Params = gin.Params{{Key: "shortUrl", Value: shortURL}}
	DeleteShortUrl(correctContext)
	assert.Equal(t, http.StatusOK, correctResponse.Code)

	_, err := store.RetrieveInitialUrl(shortURL)
	assert.ErrorIs(t, err, redis.Nil)
}

func TestGetShortUrlClicksReturnsCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shortURL := fmt.Sprintf("handler-count-%d", time.Now().UnixNano())
	ownerID := "handler-count-owner"
	require.NoError(t, store.SaveUrlMapping(shortURL, "https://example.com/count", ownerID))
	t.Cleanup(func() {
		_, _ = store.DeleteUrlMapping(shortURL, ownerID)
	})
	require.NoError(t, incrementClicks(shortURL, 2))

	context, response := newHandlerContext(http.MethodGet, "/"+shortURL+"/count", "")
	context.Params = gin.Params{{Key: "shortUrl", Value: shortURL}}
	GetShortUrlClicks(context)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"clicks":2`)
}

func TestGetShortUrlClicksReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shortURL := fmt.Sprintf("handler-missing-count-%d", time.Now().UnixNano())
	context, response := newHandlerContext(http.MethodGet, "/"+shortURL+"/count", "")
	context.Params = gin.Params{{Key: "shortUrl", Value: shortURL}}

	GetShortUrlClicks(context)

	assert.Equal(t, http.StatusNotFound, response.Code)
	assert.Contains(t, response.Body.String(), "short url not found")
}

func incrementClicks(shortURL string, count int) error {
	for range count {
		if _, err := store.IncrementClickCount(shortURL); err != nil {
			return err
		}
	}
	return nil
}
