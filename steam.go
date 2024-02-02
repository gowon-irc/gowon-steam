package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"gopkg.in/errgo.v2/fmt/errors"
)

const (
	resolveVanityUrl      = "https://api.steampowered.com/ISteamUser/ResolveVanityURL/v1/?key=%s&vanityurl=%s"
	recentlyPlayedUrl     = "https://api.steampowered.com/IPlayerService/GetRecentlyPlayedGames/v1/?key=%s&steamid=%s"
	playerAchievementsUrl = "https://api.steampowered.com/ISteamUserStats/GetPlayerAchievements/v0001/?key=%s&steamid=%s&appid=%d&format=json&l=en"
)

var (
	profileNotFoundErr  = errors.New("id not found")
	profileNotPublicErr = errors.New("profile is not public")
)

type resolveVanityURLRes struct {
	Response struct {
		SteamId string
		Success int
	}
}

func steamGetId(apiKey, user string, client *http.Client) (string, error) {
	url := fmt.Sprintf(resolveVanityUrl, apiKey, user)

	j := &resolveVanityURLRes{}

	res, err := client.Get(url)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(body, &j)
	if err != nil {
		return "", err
	}

	if j.Response.Success != 1 {
		return "", profileNotFoundErr
	}

	return j.Response.SteamId, nil
}

type recentlyPlayedRes struct {
	Response struct {
		Games []struct {
			AppId int
			Name  string
		}
	}
}

func (rpr recentlyPlayedRes) Names() (out []string) {
	out = []string{}

	for _, g := range rpr.Response.Games {
		out = append(out, g.Name)
	}
	return out
}

func (rpr recentlyPlayedRes) Ids() (out []int) {
	out = []int{}

	for _, g := range rpr.Response.Games {
		out = append(out, g.AppId)
	}
	return out
}

func getRecentlyPlayed(apiKey, id string, client *http.Client) (*recentlyPlayedRes, error) {
	url := fmt.Sprintf(recentlyPlayedUrl, apiKey, id)

	j := &recentlyPlayedRes{}

	res, err := client.Get(url)
	if err != nil {
		return j, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return j, err
	}

	err = json.Unmarshal(body, &j)
	if err != nil {
		return j, err
	}

	return j, nil
}

func colourList(in []string) (out []string) {
	out = []string{}

	colours := []string{"green", "red", "blue", "orange", "magenta", "cyan", "yellow"}
	cl := len(colours)

	for n, i := range in {
		c := colours[n%cl]
		o := fmt.Sprintf("{%s}%s{clear}", c, i)
		out = append(out, o)
	}

	return out
}

func steamLastGame(apiKey, user string, client *http.Client) (string, error) {
	id, err := steamGetId(apiKey, user, client)

	if errors.Is(profileNotFoundErr)(err) {
		return fmt.Sprintf("Error: no id found for %s", user), nil
	}

	if err != nil {
		return "", err
	}

	recentlyPlayed, err := getRecentlyPlayed(apiKey, id, client)
	if err != nil {
		return "", err
	}

	if len(recentlyPlayed.Response.Games) == 0 {
		return fmt.Sprintf("%s has no recently played steam games", user), nil
	}

	cl := colourList(recentlyPlayed.Names())

	return fmt.Sprintf("%s's recently played steam games: %s", user, strings.Join(cl, ", ")), nil
}

type playerAchievementsRes struct {
	PlayerStats struct {
		GameName     string
		Achievements []playerAchievement
		Error        string
	}
}

type playerAchievement struct {
	UnlockTime  int
	Name        string
	Description string
}

func getAchievements(apiKey, id string, appId int, client *http.Client) (*playerAchievementsRes, error) {
	url := fmt.Sprintf(playerAchievementsUrl, apiKey, id, appId)

	j := &playerAchievementsRes{}

	res, err := client.Get(url)
	if err != nil {
		return j, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return j, err
	}

	err = json.Unmarshal(body, &j)
	if err != nil {
		return j, err
	}

	if j.PlayerStats.Error == "Profile is not public" {
		return j, profileNotPublicErr
	}

	return j, nil
}

func newestAchievement(am map[string]*playerAchievementsRes) (*playerAchievementsRes, playerAchievement) {
	game := &playerAchievementsRes{}
	newest := playerAchievement{
		UnlockTime: 0,
	}

	for _, as := range am {
		for _, a := range as.PlayerStats.Achievements {
			if a.UnlockTime > newest.UnlockTime {
				game = as
				newest = a
			}
		}
	}

	return game, newest
}

func getAchievementCount(as *playerAchievementsRes) string {
	total := len(as.PlayerStats.Achievements)

	achieved := 0
	for _, a := range as.PlayerStats.Achievements {
		if a.UnlockTime > 0 {
			achieved += 1
		}
	}

	c := func(a, t int) string {
		if a == t {
			return "green"
		}

		return "yellow"
	}

	colour := c(total, achieved)

	return fmt.Sprintf("{%s}%d/%d{clear}", colour, achieved, total)
}

func steamLastAchievement(apiKey, user string, client *http.Client) (string, error) {
	id, err := steamGetId(apiKey, user, client)

	if errors.Is(profileNotFoundErr)(err) {
		return fmt.Sprintf("Error: no id found for %s", user), nil
	}

	if err != nil {
		return "", err
	}

	recentlyPlayed, err := getRecentlyPlayed(apiKey, id, client)
	if err != nil {
		return "", err
	}

	achievementsMap := make(map[string]*playerAchievementsRes)
	for _, i := range recentlyPlayed.Ids() {
		as, err := getAchievements(apiKey, id, i, client)

		if errors.Is(profileNotPublicErr)(err) {
			return "Error: profile is not public", nil
		}

		if err != nil {
			return "", err
		}

		game := as.PlayerStats.GameName
		achievementsMap[game] = as
	}

	if err != nil {
		return "", err
	}

	game, newest := newestAchievement(achievementsMap)

	if newest.UnlockTime == 0 {
		return fmt.Sprintf("%s has no recently unlocked steam achievements", user), nil
	}

	return fmt.Sprintf("%s's last steam achievement: %s - %s (%s)", user, game.PlayerStats.GameName, newest.Name, newest.Description), nil
}
