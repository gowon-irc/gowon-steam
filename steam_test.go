package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func NewConditionalTestClient(bodies map[string]string) *http.Client {
	f := func(req *http.Request) *http.Response {
		body := bodies[req.URL.String()]

		return &http.Response{
			StatusCode: 200,
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
		name     string
		testFile string
		id       string
		errMsg   string
	}{
		{
			name:     "Empty data returned",
			testFile: "empty",
			id:       "",
			errMsg:   "unexpected end of JSON input",
		},
		{
			name:     "No match",
			testFile: "no_match.json",
			id:       "",
			errMsg:   profileNotFoundErr.Error(),
		},
		{
			name:     "Success",
			testFile: "success.json",
			id:       "999",
			errMsg:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := openTestFile(t, "TestSteamGetId", tc.testFile)
			client := NewTestClient(200, string(body))

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

func TestGetRecentlyPlayed(t *testing.T) {
	cases := []struct {
		name     string
		testFile string
		errMsg   string
	}{
		{
			name:     "Empty data returned",
			testFile: "empty",
			errMsg:   "unexpected end of JSON input",
		},
		{
			name:     "No games",
			testFile: "no_games.json",
			errMsg:   "",
		},
		{
			name:     "Three games",
			testFile: "three_games.json",
			errMsg:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := openTestFile(t, "TestGetRecentlyPlayed", tc.testFile)
			client := NewTestClient(200, string(body))

			_, err := getRecentlyPlayed("key", "id", client)

			if tc.errMsg == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errMsg)
			}
		})
	}
}

func TestColourList(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		out  []string
	}{
		{
			name: "Empty list",
			in:   []string{},
			out:  []string{},
		},
		{
			name: "One string",
			in:   []string{"a"},
			out:  []string{"{green}a{clear}"},
		},
		{
			name: "Two strings",
			in:   []string{"a", "b"},
			out:  []string{"{green}a{clear}", "{red}b{clear}"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cl := colourList(tc.in)
			assert.Equal(t, tc.out, cl)
		})
	}
}

func TestColourLoop(t *testing.T) {
	in := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	out := colourList(in)

	assert.Equal(t, out[7], "{green}h{clear}")
}

func TestSteamLastGame(t *testing.T) {
	cases := []struct {
		name      string
		testFiles [2]string
		out       string
		errMsg    string
	}{
		{
			name:      "get id empty",
			testFiles: [2]string{"empty", "empty"},
			out:       "",
			errMsg:    "unexpected end of JSON input",
		},
		{
			name:      "id not found",
			testFiles: [2]string{"id_not_found.json", "empty"},
			out:       "Error: no id found for id",
			errMsg:    "",
		},
		{
			name:      "no games",
			testFiles: [2]string{"id_found.json", "no_games.json"},
			out:       "id has no recently played steam games",
			errMsg:    "",
		},
		{
			name:      "three games",
			testFiles: [2]string{"id_found.json", "three_games.json"},
			out:       "id's recently played steam games: {green}1{clear}, {red}2{clear}, {blue}3{clear}",
			errMsg:    "",
		},
	}

	rvu := fmt.Sprintf(resolveVanityUrl, "key", "id")
	rpu := fmt.Sprintf(recentlyPlayedUrl, "key", "999")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rvub := openTestFile(t, "TestSteamLastGame", tc.testFiles[0])
			rpub := openTestFile(t, "TestSteamLastGame", tc.testFiles[1])
			bodies := map[string]string{
				rvu: string(rvub),
				rpu: string(rpub),
			}
			client := NewConditionalTestClient(bodies)

			out, err := steamLastGame("key", "id", client)

			assert.Equal(t, out, tc.out)

			if tc.errMsg == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errMsg)
			}
		})
	}
}

func TestGetAchievements(t *testing.T) {
	cases := []struct {
		name     string
		testFile string
		errMsg   string
	}{
		{
			name:     "Empty data returned",
			testFile: "empty",
			errMsg:   "unexpected end of JSON input",
		},
		{
			name:     "Success",
			testFile: "success.json",
			errMsg:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := openTestFile(t, "TestGetAchievements", tc.testFile)
			client := NewTestClient(200, string(body))

			_, err := getRecentlyPlayed("key", "id", client)

			if tc.errMsg == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errMsg)
			}
		})
	}
}

func TestNewestAchievement(t *testing.T) {
	makeResMap := func(ids ...int) map[string]*playerAchievementsRes {
		rm := make(map[string]*playerAchievementsRes)

		for _, i := range ids {
			j := openTestFile(t, "TestNewestAchievement", fmt.Sprintf("%d.json", i))

			r := playerAchievementsRes{}
			err := json.Unmarshal(j, &r)
			assert.Nil(t, err)

			rm[r.PlayerStats.GameName] = &r
		}

		return rm
	}

	cases := []struct {
		name string
		m    map[string]*playerAchievementsRes
		out  string
	}{
		{
			name: "Empty map",
			m:    map[string]*playerAchievementsRes{},
			out:  "",
		},
		{
			name: "One game",
			m:    makeResMap(1),
			out:  "SUPERHOT: MIND CONTROL DELETE - MORE ()",
		},
		{
			name: "Two ids passed, newest first",
			m:    makeResMap(1, 2),
			out:  "SUPERHOT: MIND CONTROL DELETE - MORE ()",
		},
		{
			name: "Two ids passed, newest second",
			m:    makeResMap(2, 1),
			out:  "SUPERHOT: MIND CONTROL DELETE - MORE ()",
		},
		{
			name: "One id, no achievements",
			m:    makeResMap(3),
			out:  "",
		},
		{
			name: "Two ids, first no achivements",
			m:    makeResMap(3, 1),
			out:  "SUPERHOT: MIND CONTROL DELETE - MORE ()",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := newestAchievement(tc.m)

			assert.Equal(t, tc.out, out)
		})
	}
}

func TestGetAchievementCount(t *testing.T) {
	makeRes := func(id int) *playerAchievementsRes {
		j := openTestFile(t, "TestGetAchievementCount", fmt.Sprintf("%d.json", id))

		r := playerAchievementsRes{}
		err := json.Unmarshal(j, &r)
		assert.Nil(t, err)

		return &r
	}

	cases := []struct {
		name string
		id   int
		out  string
	}{
		{
			name: "Multiple achieved",
			id:   1,
			out:  "{yellow}3/14{clear}",
		},
		{
			name: "One achieved",
			id:   2,
			out:  "{yellow}1/3{clear}",
		},
		{
			name: "No achievements",
			id:   3,
			out:  "{green}0/0{clear}",
		},
		{
			name: "None achieved",
			id:   4,
			out:  "{yellow}0/3{clear}",
		},
		{
			name: "All achieved",
			id:   5,
			out:  "{green}3/3{clear}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := makeRes(tc.id)
			out := getAchievementCount(r)

			assert.Equal(t, tc.out, out)
		})
	}
}

func TestSteamLastAchievement(t *testing.T) {
	cases := []struct {
		name      string
		testFiles [3]string
		out       string
		errMsg    string
	}{
		{
			name:      "get id empty",
			testFiles: [3]string{"empty", "empty", "empty"},
			out:       "",
			errMsg:    "unexpected end of JSON input",
		},
		{
			name:      "id not found",
			testFiles: [3]string{"id_not_found.json", "empty", "empty"},
			out:       "Error: no id found for id",
			errMsg:    "",
		},
		{
			name:      "id found, recently played empty",
			testFiles: [3]string{"id_found.json", "empty", "empty"},
			out:       "",
			errMsg:    "unexpected end of JSON input",
		},
		{
			name:      "no recently played games",
			testFiles: [3]string{"id_found.json", "no_games.json", "empty"},
			out:       "id has no recently unlocked steam achievements",
			errMsg:    "",
		},
		{
			name:      "get achievements empty",
			testFiles: [3]string{"id_found.json", "one_game.json", "empty"},
			out:       "",
			errMsg:    "unexpected end of JSON input",
		},
		{
			name:      "achievements found",
			testFiles: [3]string{"id_found.json", "one_game.json", "achievements.json"},
			out:       "id's last steam achievement: SUPERHOT: MIND CONTROL DELETE - MORE ()",
			errMsg:    "",
		},
	}

	rvu := fmt.Sprintf(resolveVanityUrl, "key", "id")
	rpu := fmt.Sprintf(recentlyPlayedUrl, "key", "999")
	pau := fmt.Sprintf(playerAchievementsUrl, "key", "999", 999)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rvub := openTestFile(t, "TestSteamLastAchievement", tc.testFiles[0])
			rpub := openTestFile(t, "TestSteamLastAchievement", tc.testFiles[1])
			paub := openTestFile(t, "TestSteamLastAchievement", tc.testFiles[2])
			bodies := map[string]string{
				rvu: string(rvub),
				rpu: string(rpub),
				pau: string(paub),
			}
			client := NewConditionalTestClient(bodies)

			out, err := steamLastAchievement("key", "id", client)

			assert.Equal(t, out, tc.out)

			if tc.errMsg == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errMsg)
			}
		})
	}
}
