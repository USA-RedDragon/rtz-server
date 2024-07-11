package apis

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/USA-RedDragon/rtz-server/internal/utils"
)

type GoogleUserResponse struct {
	ID string `json:"id"`
}

func GetGoogleUserID(ctx context.Context, token string) (string, error) {
	resp, err := utils.HTTPRequest(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v1/userinfo?alt=json&access_token="+token, nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	response := GoogleUserResponse{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}
	return response.ID, nil
}
