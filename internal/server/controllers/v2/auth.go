package v2

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/USA-RedDragon/rtz-server/internal/apis"
	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	v2 "github.com/USA-RedDragon/rtz-server/internal/server/apimodels/v2"
	"github.com/USA-RedDragon/rtz-server/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/mattn/go-nulltype"
	"gorm.io/gorm"
)

func POSTAuth(c *gin.Context) {
	var data v2.POSTAuthRequest

	data.Provider = c.PostForm("provider")
	if data.Provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}
	data.Code = c.PostForm("code")
	if data.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		slog.Error("Failed to get db from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	config, ok := c.MustGet("config").(*config.Config)
	if !ok {
		slog.Error("Failed to get config from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	var user models.User
	switch data.Provider {
	case "g":
		if !config.Auth.Google.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Google auth is disabled"})
			return
		}
		user, ok = googleLogin(c, db, config, data.Code)
	case "h":
		if !config.Auth.GitHub.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "GitHub auth is disabled"})
			return
		}
		user, ok = githubLogin(c, db, config, data.Code)
	case "c":
		if !config.Auth.Custom.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Custom auth is disabled"})
			return
		}
		user, ok = customLogin(c, db, config, data.Code)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is invalid"})
		return
	}
	if !ok {
		return
	}

	token, err := utils.GenerateJWT(config.JWT.Secret, user.ID)
	if err != nil {
		slog.Error("Failed to generate JWT", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": token})
}

// exchangeToken posts body to tokenURL and decodes the JSON reply into out.
// On failure it writes the error response and returns false.
func exchangeToken(c *gin.Context, tokenURL string, body url.Values, headers map[string]string, out any) bool {
	resp, err := utils.HTTPRequest(c, http.MethodPost, tokenURL, strings.NewReader(body.Encode()), headers)
	if err != nil {
		slog.Error("Failed to make request", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Failed to get token", "status", resp.StatusCode)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return false
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		slog.Error("Failed to decode response", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return false
	}
	return true
}

// loginOrRegister finds the user, creating newUser first when it doesn't
// exist and registration is enabled. On failure it writes the error response
// and returns false.
func loginOrRegister(c *gin.Context, db *gorm.DB, config *config.Config, find func() (models.User, error), newUser models.User) (models.User, bool) {
	user, err := find()
	if err == nil {
		return user, true
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) || !config.Registration.Enabled {
		slog.Error("Failed to register or login user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return models.User{}, false
	}
	if err := db.Create(&newUser).Error; err != nil {
		slog.Error("Failed to create user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return models.User{}, false
	}
	user, err = find()
	if err != nil {
		slog.Error("Failed to find user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return models.User{}, false
	}
	return user, true
}

func googleLogin(c *gin.Context, db *gorm.DB, config *config.Config, code string) (models.User, bool) {
	urldata := url.Values{}
	urldata.Set("code", code)
	urldata.Set("client_id", config.Auth.Google.ClientID)
	urldata.Set("client_secret", config.Auth.Google.ClientSecret)
	urldata.Set("redirect_uri", config.HTTP.BackendURL+"/v2/auth/g/redirect/")
	urldata.Set("grant_type", "authorization_code")

	tokenResponse := v2.GoogleTokenResponse{}
	if !exchangeToken(c, "https://oauth2.googleapis.com/token", urldata, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, &tokenResponse) {
		return models.User{}, false
	}

	id, err := apis.GetGoogleUserID(c, tokenResponse.AccessToken)
	if err != nil {
		slog.Error("Failed to get Google user ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return models.User{}, false
	}

	return loginOrRegister(c, db, config, func() (models.User, error) {
		return models.FindUserByGoogleID(db, id)
	}, models.User{GoogleUserID: nulltype.NullStringOf(id)})
}

func githubTokenURL(urldata url.Values) string {
	return "https://github.com/login/oauth/access_token?" + urldata.Encode()
}

func githubLogin(c *gin.Context, db *gorm.DB, config *config.Config, code string) (models.User, bool) {
	urldata := url.Values{}
	urldata.Set("code", code)
	urldata.Set("client_id", config.Auth.GitHub.ClientID)
	urldata.Set("client_secret", config.Auth.GitHub.ClientSecret)

	tokenResponse := v2.GitHubTokenResponse{}
	if !exchangeToken(c, githubTokenURL(urldata), urldata, map[string]string{
		"Accept": "application/json",
	}, &tokenResponse) {
		return models.User{}, false
	}

	id, err := apis.GetGitHubUserID(c, tokenResponse.AccessToken)
	if err != nil {
		slog.Error("Failed to get GitHub user ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return models.User{}, false
	}

	return loginOrRegister(c, db, config, func() (models.User, error) {
		return models.FindUserByGitHubID(db, id)
	}, models.User{GitHubUserID: nulltype.NullInt64Of(int64(id))})
}

func customLogin(c *gin.Context, db *gorm.DB, config *config.Config, code string) (models.User, bool) {
	urldata := url.Values{}
	urldata.Set("code", code)
	urldata.Set("client_id", config.Auth.Custom.ClientID)
	urldata.Set("client_secret", config.Auth.Custom.ClientSecret)
	urldata.Set("grant_type", "authorization_code")
	urldata.Set("scope", "user:email")
	urldata.Set("redirect_uri", config.HTTP.BackendURL+"/v2/auth/c/redirect/")

	tokenResponse := v2.GitHubTokenResponse{}
	if !exchangeToken(c, config.Auth.Custom.TokenURL, urldata, map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/x-www-form-urlencoded",
	}, &tokenResponse) {
		return models.User{}, false
	}

	id, err := apis.GetCustomUserID(c, config.Auth.Custom.UserURL, tokenResponse.AccessToken)
	if err != nil {
		slog.Error("Failed to get custom user ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return models.User{}, false
	}

	return loginOrRegister(c, db, config, func() (models.User, error) {
		return models.FindUserByCustomID(db, id)
	}, models.User{CustomUserID: nulltype.NullInt64Of(int64(id))})
}

func GETAuthRedirect(c *gin.Context) {
	provider, ok := c.Params.Get("provider")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}

	queryState := c.Query("state")
	if queryState == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state is required"})
		return
	}
	// We expect state to be `service,$frontend_host`
	stateParts := strings.Split(queryState, ",")
	if len(stateParts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state is invalid"})
		return
	}

	if stateParts[0] != "service" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state is invalid"})
		return
	}

	returnHostname := stateParts[1]

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	config, ok := c.MustGet("config").(*config.Config)
	if !ok {
		slog.Error("Failed to get config from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	referer, err := url.Parse("https://" + returnHostname)
	if err != nil {
		slog.Error("Failed to parse referer", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Referer header is invalid"})
		return
	}

	authRedirect := referer.JoinPath("/auth/")
	if authRedirect == nil {
		slog.Error("Failed to join path", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Referer header is invalid"})
		return
	}

	queries := url.Values{}
	queries.Add("code", code)

	switch provider {
	case "g":
		// Google
		if !config.Auth.Google.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Google auth is disabled"})
			return
		}
		queryError := c.Query("error")
		if queryError != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": queryError})
			return
		}

		scope := c.Query("scope")
		if scope == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "scope is required"})
			return
		}
		if !strings.Contains(scope, "https://www.googleapis.com/auth/userinfo.email") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "scope is invalid"})
			return
		}
		queries.Add("provider", "g")
	case "h":
		// GitHub
		if !config.Auth.GitHub.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "GitHub auth is disabled"})
			return
		}
		queries.Add("provider", "h")
	case "c":
		// Custom
		if !config.Auth.Custom.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Custom auth is disabled"})
			return
		}
		queries.Add("provider", "c")
	}

	authRedirect.RawQuery = queries.Encode()

	// Redirect to the app with the code
	c.Redirect(http.StatusFound, authRedirect.String())
}
