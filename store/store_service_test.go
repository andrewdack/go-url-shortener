package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testStoreService = &StorageService{}

func init() {
	testStoreService = InitializeStore()
}

func TestStoreInit(t *testing.T) {
	assert.True(t, testStoreService.redisClient != nil)
}

func TestInsertionAndRetrieval(t *testing.T) {
	initialLink := "https://www.guru3d.com/news-story/spotted-ryzen-threadripper-pro-3995wx-processor-with-8-channel-ddr4,2.html"
	userUUId := "e0dba740-fc4b-4977-872c-d360239e6b1a"
	shortURL := "Jsz4k57oAX"

	// Persist data mapping
	err := SaveUrlMapping(shortURL, initialLink, userUUId)
	assert.NoError(t, err)

	// Retrieve initial URL
	retrievedUrl, err := RetrieveInitialUrl(shortURL)
	assert.NoError(t, err)

	assert.Equal(t, initialLink, retrievedUrl)
}

func TestUpdateUrlMappingRequiresMatchingUserId(t *testing.T) {
	shortURL := "update-owner-test"
	originalURL := "https://example.com/original"
	updatedURL := "https://example.com/updated"
	ownerID := "owner-user-id"

	t.Cleanup(func() {
		testStoreService.redisClient.Del(ctx, shortURL, ownerKey(shortURL))
	})

	err := SaveUrlMapping(shortURL, originalURL, ownerID)
	assert.NoError(t, err)

	_, err = UpdateUrlMapping(shortURL, updatedURL, "different-user-id")
	assert.ErrorIs(t, err, ErrUserIdMismatch)

	retrievedURL, err := RetrieveInitialUrl(shortURL)
	assert.NoError(t, err)
	assert.Equal(t, originalURL, retrievedURL)

	result, err := UpdateUrlMapping(shortURL, updatedURL, ownerID)
	assert.NoError(t, err)
	assert.Equal(t, updatedURL, result)
}
