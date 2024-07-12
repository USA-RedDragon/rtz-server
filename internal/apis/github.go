package apis

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/USA-RedDragon/rtz-server/internal/utils"
)

type UserResponse struct {
	ID int `json:"id"`
}

func GetGitHubUserID(ctx context.Context, token string) (int, error) {
	resp, err := utils.HTTPRequest(ctx, http.MethodGet, "https://api.github.com/user", nil, map[string]string{
		"Authorization":        "Bearer " + token,
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	response := UserResponse{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return 0, err
	}
	return response.ID, nil
}

func GetCustomUserID(ctx context.Context, url, token string) (int, error) {
	resp, err := utils.HTTPRequest(ctx, http.MethodGet, url, nil, map[string]string{
		"Authorization":        "Bearer " + token,
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	response := UserResponse{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return 0, err
	}
	return response.ID, nil
}
