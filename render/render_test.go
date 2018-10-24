package render

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hooklift/assert"
)

func TestRenderJSON(t *testing.T) {
	tests := []struct {
		desc            string
		data            interface{}
		code            int
		expectedHeaders http.Header
		expectedBody    []byte
	}{
		{
			"200 responses with nil body, cacheable",
			nil,
			http.StatusOK,
			http.Header{
				"Content-Type":   {"application/json; charset=UTF-8"},
				"Content-Length": {"5"},
			},
			[]byte("\"OK\"\n"),
		},
		{
			"200 responses with defined body, cacheable",
			map[string]string{"blah": "foo"},
			http.StatusOK,
			http.Header{
				"Content-Type":   {"application/json; charset=UTF-8"},
				"Content-Length": {"15"},
			},
			[]byte("{\"blah\":\"foo\"}\n"),
		},
		{
			"400, 500 responses with nil body, no cacheable",
			nil,
			http.StatusBadRequest,
			http.Header{
				"Cache-Control":  {"no-store"},
				"Content-Type":   {"application/json; charset=UTF-8"},
				"Expires":        {"0"},
				"Pragma":         {"no-cache"},
				"Content-Length": {"14"},
			},
			[]byte("\"Bad Request\"\n"),
		},
		{
			"400, 500 responses with defined body, no cacheable",
			map[string]string{"blah": "foo"},
			http.StatusBadRequest,
			http.Header{
				"Cache-Control":  {"no-store"},
				"Content-Type":   {"application/json; charset=UTF-8"},
				"Expires":        {"0"},
				"Pragma":         {"no-cache"},
				"Content-Length": {"15"},
			},
			[]byte("{\"blah\":\"foo\"}\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				JSON(w, tt.data, tt.code)
			}))
			defer ts.Close()
			client := ts.Client()

			res, err := client.Get(ts.URL)
			assert.Ok(t, err)

			// Do no test Date header
			res.Header.Del("date")

			assert.Equals(t, tt.code, res.StatusCode)
			assert.Equals(t, tt.expectedHeaders, res.Header)

			data, err := ioutil.ReadAll(res.Body)
			assert.Ok(t, err)
			t.Logf("body => %s", data)
			assert.Equals(t, tt.expectedBody, data)
		})
	}
}
