package handler

// import gin
import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/andrewdack/go-url-shortener/shortener"
	"github.com/andrewdack/go-url-shortener/store"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// Define Handler wrapper over storage service
type Handler struct {
	store *store.StorageService
}
// Request model definition
type UrlCreationRequest struct {
	// Public fields cuz capitalized fields
	// The backtick text after each field is a struct tag
	LongUrl string `json:"longUrl" binding:"required,url"`
	UserId  string `json:"userId" binding:"required"`
}

type UrlUpdateRequest struct {
	LongUrl string `json:"longUrl" binding:"required,url"`
	UserId  string `json:"userId" binding:"required"`
}

type UrlDeletionRequest struct {
	UserId string `json:"userId" binding:"required"`
}

// Create a New Handler wrapper over the storage service
func NewHandler(storageService *store.StorageService) *Handler {
	return &Handler{
		store: storageService,
	}
}

func (h *Handler) Store() *store.StorageService {
	return h.store
}

func (h *Handler) CreateShortUrl(c *gin.Context) {
	var createShortUrlRequest UrlCreationRequest

	// Read the request body, parse the JSON, and bind it to the struct
	// If the request body is not valid JSON or doesn't match the struct, return a 400 Bad Request response
	if err := c.ShouldBindJSON(&createShortUrlRequest); err != nil {
		// Return a 400 Bad Request response with the error message
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shortURL := shortener.GenerateShortLink(createShortUrlRequest.LongUrl, createShortUrlRequest.UserId)
	if err := h.store.SaveUrlMapping(c.Request.Context(), shortURL, createShortUrlRequest.LongUrl, createShortUrlRequest.UserId); err != nil {
		log.Printf("failed to save short url mapping: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save short url"})
		return
	}

	host := os.Getenv("PUBLIC_HOST")
	if host == "" {
		// If PUBLIC_HOST is not set, default to localhost with port 9808
		host = "http://localhost:9808/"
	}
	c.JSON(200, gin.H{
		"message":  "Short URL created successfully",
		"shortUrl": host + shortURL,
	})
}

func (h *Handler) HandleShortUrlRedirect(c *gin.Context) {
	shortUrl := c.Param("shortUrl")
	initialUrl, err := h.store.RetrieveInitialUrl(c.Request.Context(), shortUrl)
	if errors.Is(err, redis.Nil) {
		c.JSON(http.StatusNotFound, gin.H{"error": "short url not found"})
		return
	}
	if err != nil {
		log.Printf("failed to retrieve short url %s: %v", shortUrl, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve short url"})
		return
	}
	c.Redirect(http.StatusFound, initialUrl)
}

func (h *Handler) UpdateShortUrl(c *gin.Context) {
	shortUrl := c.Param("shortUrl")

	var updateShortUrlRequest UrlUpdateRequest
	if err := c.ShouldBindJSON(&updateShortUrlRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedUrl, err := h.store.UpdateUrlMapping(c.Request.Context(), shortUrl, updateShortUrlRequest.LongUrl, updateShortUrlRequest.UserId)
	if errors.Is(err, redis.Nil) {
		c.JSON(http.StatusNotFound, gin.H{"error": "short url not found"})
		return
	}
	if errors.Is(err, store.ErrUserIdMismatch) {
		c.JSON(http.StatusForbidden, gin.H{"error": "user does not own short url"})
		return
	}
	if err != nil {
		log.Printf("failed to update short url %s: %v", shortUrl, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update short url"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Short URL updated successfully",
		"shortUrl": shortUrl,
		"longUrl":  updatedUrl,
	})
}

func (h *Handler) DeleteShortUrl(c *gin.Context) {
	shortUrl := c.Param("shortUrl")

	var deleteShortUrlRequest UrlDeletionRequest
	if err := c.ShouldBindJSON(&deleteShortUrlRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	deletedUrl, err := h.store.DeleteUrlMapping(c.Request.Context(), shortUrl, deleteShortUrlRequest.UserId)
	if errors.Is(err, redis.Nil) {
		c.JSON(http.StatusNotFound, gin.H{"error": "short url not found"})
		return
	}
	if errors.Is(err, store.ErrUserIdMismatch) {
		c.JSON(http.StatusForbidden, gin.H{"error": "user does not own short url"})
		return
	}
	if err != nil {
		log.Printf("failed to delete short url %s: %v", shortUrl, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete short url"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Short URL deleted successfully",
		"shortUrl": deletedUrl,
	})
}

func (h *Handler) GetShortUrlClicks(c *gin.Context) {
	shortUrl := c.Param("shortUrl")

	count, err := h.store.RetrieveClickCount(c.Request.Context(), shortUrl)
	if errors.Is(err, redis.Nil) {
		c.JSON(http.StatusNotFound, gin.H{"error": "short url not found"})
		return
	}
	if err != nil {
		log.Printf("failed to retrieve click count for %s: %v", shortUrl, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve click count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"shortUrl": shortUrl,
		"clicks":   count,
	})
}
