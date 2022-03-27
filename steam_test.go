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

func generateRecentlyPlayedRes(count int) recentlyPlayedRes {
	r := recentlyPlayedRes{}

	for i := 0; i < count; i++ {
		g := struct {
			AppId int
			Name  string
		}{
			AppId: 1,
			Name:  "game",
		}

		r.Response.Games = append(r.Response.Games, g)
	}

	return r
}

func TestRecentlyPlayedResNames(t *testing.T) {
	cases := []struct {
		name  string
		count int
		out   []string
	}{
		{
			name:  "No games",
			count: 0,
			out:   []string{},
		},
		{
			name:  "One game",
			count: 1,
			out:   []string{"game"},
		},
		{
			name:  "Two games",
			count: 2,
			out:   []string{"game", "game"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rps := generateRecentlyPlayedRes(tc.count)
			out := rps.Names()

			assert.Equal(t, tc.out, out)
		})
	}
}

func TestRecentlyPlayedResIds(t *testing.T) {
	cases := []struct {
		name  string
		count int
		out   []int
	}{
		{
			name:  "No games",
			count: 0,
			out:   []int{},
		},
		{
			name:  "One game",
			count: 1,
			out:   []int{1},
		},
		{
			name:  "Two games",
			count: 2,
			out:   []int{1, 1},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rps := generateRecentlyPlayedRes(tc.count)
			out := rps.Ids()

			assert.Equal(t, tc.out, out)
		})
	}
}
