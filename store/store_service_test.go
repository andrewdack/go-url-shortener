package store

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testStoreService *StorageService

func TestMain(m *testing.M) {
	service, err := NewStore(context.Background())
	if err != nil {
		panic(err)
	}
	testStoreService = service
	code := m.Run()
	_ = service.Close()
	os.Exit(code)
}

func TestStoreInit(t *testing.T) {
	assert.NotNil(t, testStoreService)
	assert.NotNil(t, testStoreService.redisClient)
	assert.NotNil(t, testStoreService.pgPool)
}

func TestInsertionAndRetrieval(t *testing.T) {
	ctx := context.Background()
	initialLink := "https://www.guru3d.com/news-story/spotted-ryzen-threadripper-pro-3995wx-processor-with-8-channel-ddr4,2.html"
	userID := "e0dba740-fc4b-4977-872c-d360239e6b1a"
	shortURL := "Jsz4k57oAX"

	t.Cleanup(func() {
		_, _ = testStoreService.DeleteUrlMapping(ctx, shortURL, userID)
	})

	err := testStoreService.SaveUrlMapping(ctx, shortURL, initialLink, userID)
	require.NoError(t, err)

	retrievedURL, err := testStoreService.RetrieveInitialUrl(ctx, shortURL)
	require.NoError(t, err)
	assert.Equal(t, initialLink, retrievedURL)
}

func TestUpdateUrlMappingRequiresMatchingUserID(t *testing.T) {
	ctx := context.Background()
	shortURL := "update-owner-test"
	originalURL := "https://example.com/original"
	updatedURL := "https://example.com/updated"
	ownerID := "owner-user-id"

	t.Cleanup(func() {
		_, _ = testStoreService.DeleteUrlMapping(ctx, shortURL, ownerID)
	})

	require.NoError(t, testStoreService.SaveUrlMapping(ctx, shortURL, originalURL, ownerID))

	_, err := testStoreService.UpdateUrlMapping(ctx, shortURL, updatedURL, "different-user-id")
	assert.ErrorIs(t, err, ErrUserIdMismatch)

	retrievedURL, err := testStoreService.RetrieveInitialUrl(ctx, shortURL)
	require.NoError(t, err)
	assert.Equal(t, originalURL, retrievedURL)

	result, err := testStoreService.UpdateUrlMapping(ctx, shortURL, updatedURL, ownerID)
	require.NoError(t, err)
	assert.Equal(t, updatedURL, result)
}

func TestClickCountRequiresExistingShortURL(t *testing.T) {
	ctx := context.Background()
	shortURL := "missing-click-count-test"

	_, err := testStoreService.RetrieveClickCount(ctx, shortURL)
	assert.ErrorIs(t, err, ErrNotFound)
}
