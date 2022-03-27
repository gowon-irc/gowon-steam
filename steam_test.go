package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(statusCode int, body string) *http.Client {
	f := func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: statusCode,
			Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
			Header:     make(http.Header),
		}
	}

	return &http.Client{
		Transport: RoundTripFunc(f),
	}
}

func openTestFile(t *testing.T, test, filename string) []byte {
	fp := filepath.Join("testdata", test, filename)
	out, err := ioutil.ReadFile(fp)

	if err != nil {
		t.Fatalf("failed to read test file: %s", err)
	}

	return out
}

func TestSteamGetId(t *testing.T) {
	cases := []struct {
		name       string
		testFile   string
		statusCode int
		id         string
		errMsg     string
	}{
		{
			name:       "Empty data returned",
			testFile:   "empty",
			statusCode: 200,
			id:         "",
			errMsg:     "unexpected end of JSON input",
		},
		{
			name:       "No match",
			testFile:   "no_match.json",
			statusCode: 200,
			id:         "",
			errMsg:     profileNotFoundErr.Error(),
		},
		{
			name:       "Success",
			testFile:   "success.json",
			statusCode: 200,
			id:         "999",
			errMsg:     "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := openTestFile(t, "TestSteamGetId", tc.testFile)
			client := NewTestClient(tc.statusCode, string(body))

			id, err := steamGetId("key", "user", client)

			if tc.errMsg == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errMsg)
			}

			assert.Equal(t, id, tc.id)
		})
	}
}
