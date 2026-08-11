package handler

// import gin
import (
	"errors"
	"net/http"

	"github.com/andrewdack/go-url-shortener/shortener"
	"github.com/andrewdack/go-url-shortener/store"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// Request model definition
type UrlCreationRequest struct {
	// Public fields cuz capitalized fields
	// The backtick text after each field is a struct tag
	LongUrl string `json:"longUrl" binding:"required"`
	UserId  string `json:"userId" binding:"required"`
}

func CreateShortUrl(c *gin.Context) {
	var createShortUrlRequest UrlCreationRequest

	// Read the request body, parse the JSON, and bind it to the struct
	// If the request body is not valid JSON or doesn't match the struct, return a 400 Bad Request response
	if err := c.ShouldBindJSON(&createShortUrlRequest); err != nil {
		// Return a 400 Bad Request response with the error message
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shortURL := shortener.GenerateShortLink(createShortUrlRequest.LongUrl, createShortUrlRequest.UserId)
	if err := store.SaveUrlMapping(shortURL, createShortUrlRequest.LongUrl, createShortUrlRequest.UserId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save short url"})
		return
	}

	host := "http://localhost:9808/"
	c.JSON(200, gin.H{
		"message": "Short URL created successfully",
		"shortUrl": host + shortURL,
	})
}

func HandleShortUrlRedirect(c *gin.Context) {
	shortUrl := c.Param("shortUrl")
	initialUrl, err := store.RetrieveInitialUrl(shortUrl)
	if errors.Is(err, redis.Nil) {
		c.JSON(http.StatusNotFound, gin.H{"error": "short url not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve short url"})
		return
	}
	c.Redirect(http.StatusFound, initialUrl)
}