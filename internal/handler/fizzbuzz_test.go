package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/houssemlou/fizz/internal/domain/fizzbuzz"
	"github.com/houssemlou/fizz/internal/handler"
	"github.com/houssemlou/fizz/internal/mocks"
)

const testAPIKey = "test-key"

func init() {
	gin.SetMode(gin.TestMode)
}

func newServer(t *testing.T, svc *mocks.MockService) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(handler.New(":0", "prod", testAPIKey, handler.NewHandler(svc)).Handler())
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, ts *httptest.Server, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	require.NoError(t, err)
	req.Header.Set("X-API-Key", testAPIKey)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func TestUnauthorized(t *testing.T) {
	ts := newServer(t, mocks.NewMockService(t))

	resp, err := http.Get(ts.URL + "/v1/health")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestFizzBuzz(t *testing.T) {
	want := []string{
		"1", "2", "fizz", "4", "buzz",
		"fizz", "7", "8", "fizz", "buzz",
		"11", "fizz", "13", "14", "fizzbuzz",
	}

	svc := mocks.NewMockService(t)
	svc.EXPECT().
		Generate(mock.Anything, mock.AnythingOfType("fizzbuzz.Params")).
		Return(want, nil)

	ts := newServer(t, svc)
	resp := get(t, ts, "/v1/fizzbuzz?int1=3&int2=5&limit=15&str1=fizz&str2=buzz")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Result []string `json:"result"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, want, body.Result)
}

func TestFizzBuzzValidation(t *testing.T) {
	cases := []struct {
		name string
		url  string
		code int
	}{
		{"missing int1", "/v1/fizzbuzz?int2=5&limit=15&str1=fizz&str2=buzz", http.StatusBadRequest},
		{"invalid int1", "/v1/fizzbuzz?int1=abc&int2=5&limit=15&str1=fizz&str2=buzz", http.StatusBadRequest},
		{"zero int1", "/v1/fizzbuzz?int1=0&int2=5&limit=15&str1=fizz&str2=buzz", http.StatusBadRequest},
		{"empty str1", "/v1/fizzbuzz?int1=3&int2=5&limit=15&str1=&str2=buzz", http.StatusBadRequest},
		{"limit too large", "/v1/fizzbuzz?int1=3&int2=5&limit=1001&str1=fizz&str2=buzz", http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := newServer(t, mocks.NewMockService(t))
			resp := get(t, ts, tc.url)
			resp.Body.Close()
			assert.Equal(t, tc.code, resp.StatusCode)
		})
	}
}

func TestStats(t *testing.T) {
	p1 := fizzbuzz.Params{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}
	top := &fizzbuzz.TopResult{Params: p1, Hits: 2}

	svc := mocks.NewMockService(t)
	svc.EXPECT().
		Generate(mock.Anything, mock.AnythingOfType("fizzbuzz.Params")).
		Return([]string{"1"}, nil).
		Times(3)
	svc.EXPECT().Top(mock.Anything).Return(nil, nil).Once()
	svc.EXPECT().Top(mock.Anything).Return(top, nil).Once()

	ts := newServer(t, svc)

	resp := get(t, ts, "/v1/stats")
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	for range 2 {
		r := get(t, ts, "/v1/fizzbuzz?int1=3&int2=5&limit=15&str1=fizz&str2=buzz")
		r.Body.Close()
	}
	r := get(t, ts, "/v1/fizzbuzz?int1=2&int2=7&limit=50&str1=foo&str2=bar")
	r.Body.Close()

	resp = get(t, ts, "/v1/stats")
	defer resp.Body.Close()

	var body struct {
		Request struct {
			Int1 int    `json:"int1"`
			Str1 string `json:"str1"`
		} `json:"request"`
		Hits int `json:"hits"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, 2, body.Hits)
	assert.Equal(t, 3, body.Request.Int1)
	assert.Equal(t, "fizz", body.Request.Str1)
}

func TestHealth(t *testing.T) {
	ts := newServer(t, mocks.NewMockService(t))
	resp := get(t, ts, "/v1/health")
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
