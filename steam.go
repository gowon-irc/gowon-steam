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

func steamGetId(apiKey, user string) (string, error) {
	url := fmt.Sprintf(resolveVanityUrl, apiKey, user)

	j := &resolveVanityURLRes{}

	res, err := http.Get(url)
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
	for _, g := range rpr.Response.Games {
		out = append(out, g.Name)
	}
	return out
}

func (rpr recentlyPlayedRes) Ids() (out []int) {
	for _, g := range rpr.Response.Games {
		out = append(out, g.AppId)
	}
	return out
}

func getRecentlyPlayed(apiKey, id string) (*recentlyPlayedRes, error) {
	url := fmt.Sprintf(recentlyPlayedUrl, apiKey, id)

	j := &recentlyPlayedRes{}

	res, err := http.Get(url)
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
	colours := []string{"green", "red", "blue", "orange", "magenta", "cyan", "yellow"}
	cl := len(colours)

	for n, i := range in {
		c := colours[n%cl]
		o := fmt.Sprintf("{%s}%s{clear}", c, i)
		out = append(out, o)
	}

	return out
}

func steamLastGame(apiKey, user string) (string, error) {
	id, err := steamGetId(apiKey, user)

	if errors.Is(profileNotFoundErr)(err) {
		return fmt.Sprintf("Error: no id found for %s", user), nil
	}

	if err != nil {
		return "", err
	}

	recentlyPlayed, err := getRecentlyPlayed(apiKey, id)
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

func getAchievements(apiKey, id string, appId int) (*playerAchievementsRes, error) {
	url := fmt.Sprintf(playerAchievementsUrl, apiKey, id, appId)

	j := &playerAchievementsRes{}

	res, err := http.Get(url)
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

func newestAchievement(apiKey, id string, appIds []int) (string, error) {
	game := ""
	newest := playerAchievement{
		UnlockTime: 0,
	}

	for _, i := range appIds {
		as, err := getAchievements(apiKey, id, i)
		if err != nil {
			return "", err
		}

		if as.PlayerStats.Error == "Profile is not public" {
			return "", profileNotPublicErr
		}

		for _, a := range as.PlayerStats.Achievements {
			if a.UnlockTime > newest.UnlockTime {
				game = as.PlayerStats.GameName
				newest = a
			}
		}
	}

	if newest.UnlockTime == 0 {
		return "", nil
	}

	return fmt.Sprintf("%s (%s) - %s", newest.Name, newest.Description, game), nil
}

func steamLastAchievement(apiKey, user string) (string, error) {
	id, err := steamGetId(apiKey, user)

	if errors.Is(profileNotFoundErr)(err) {
		return fmt.Sprintf("Error: no id found for %s", user), nil
	}

	if err != nil {
		return "", err
	}

	recentlyPlayed, err := getRecentlyPlayed(apiKey, id)
	if err != nil {
		return "", err
	}

	n, err := newestAchievement(apiKey, id, recentlyPlayed.Ids())

	if errors.Is(profileNotPublicErr)(err) {
		return "Error: profile is not public", nil
	}

	if err != nil {
		return "", err
	}

	if n == "" {
		return fmt.Sprintf("%s has no recently unlocked steam achievements", user), nil
	}

	return fmt.Sprintf("%s's last steam achievement: %s", user, n), nil
}
