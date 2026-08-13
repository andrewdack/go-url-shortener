package router

import (
	stdcontext "context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/andrewdack/go-url-shortener/handler"
	"github.com/andrewdack/go-url-shortener/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testHandler *handler.Handler

func TestMain(m *testing.M) {
	service, err := store.NewStore(stdcontext.Background())
	if err != nil {
		panic(err)
	}
	testHandler = handler.NewHandler(service)
	code := m.Run()
	_ = service.Close()
	os.Exit(code)
}

func TestSetupRouterHealthCheck(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	response := httptest.NewRecorder()

	SetupRouter(testHandler).ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "http://localhost:5173", response.Header().Get("Access-Control-Allow-Origin"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "OK", body["message"])
}

func TestSetupRouterHandlesFrontendPreflights(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		path   string
		method string
	}{
		{name: "create", path: "/create-short-url", method: http.MethodPost},
		{name: "update", path: "/short-code", method: http.MethodPatch},
		{name: "delete", path: "/short-code", method: http.MethodDelete},
		{name: "click count", path: "/short-code/count", method: http.MethodGet},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodOptions, testCase.path, nil)
			request.Header.Set("Origin", "http://localhost:5173")
			request.Header.Set("Access-Control-Request-Method", testCase.method)
			request.Header.Set("Access-Control-Request-Headers", "content-type")
			response := httptest.NewRecorder()

			SetupRouter(testHandler).ServeHTTP(response, request)

			assert.Equal(t, http.StatusNoContent, response.Code)
			assert.Equal(t, "http://localhost:5173", response.Header().Get("Access-Control-Allow-Origin"))
			assert.Contains(t, response.Header().Get("Access-Control-Allow-Methods"), testCase.method)
		})
	}
}

func TestSetupRouterRegistersExpectedRoutes(t *testing.T) {
	t.Parallel()

	routes := SetupRouter(testHandler).Routes()
	registered := make(map[string]bool, len(routes))
	for _, route := range routes {
		registered[route.Method+" "+route.Path] = true
	}

	expected := []string{
		"GET /health",
		"POST /create-short-url",
		"GET /:shortUrl",
		"GET /:shortUrl/count",
		"PATCH /:shortUrl",
		"DELETE /:shortUrl",
	}

	for _, route := range expected {
		assert.Truef(t, registered[route], "expected route %s to be registered", route)
	}
}
