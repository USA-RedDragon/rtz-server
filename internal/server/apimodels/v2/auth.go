package v2

type POSTAuthRequest struct {
	Provider string
	Code     string
}

type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
}
