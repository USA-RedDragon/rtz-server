package v2

import (
	"encoding/json"
	"errors"
	"fmt"
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
		//nolint:golint,gosec
		tokenURL := "https://oauth2.googleapis.com/token"

		urldata := url.Values{}
		urldata.Set("code", data.Code)
		urldata.Set("client_id", config.Auth.Google.ClientID)
		urldata.Set("client_secret", config.Auth.Google.ClientSecret)
		urldata.Set("redirect_uri", config.HTTP.BackendURL+"/v2/auth/g/redirect/")
		urldata.Set("grant_type", "authorization_code")

		resp, err := utils.HTTPRequest(c, http.MethodPost, tokenURL, strings.NewReader(urldata.Encode()), map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		})
		if err != nil {
			slog.Error("Failed to make request", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slog.Error("Failed to get token", "status", resp.StatusCode)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		tokenResponse := v2.GoogleTokenResponse{}

		err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
		if err != nil {
			slog.Error("Failed to decode response", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		id, err := apis.GetGoogleUserID(c, tokenResponse.AccessToken)
		if err != nil {
			slog.Error("Failed to get Google user ID", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		user, err = models.FindUserByGoogleID(db, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) && config.Registration.Enabled {
				// Create user
				err = db.Create(&models.User{
					GoogleUserID: nulltype.NullStringOf(id),
				}).Error
				if err != nil {
					slog.Error("Failed to create user", "error", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
					return
				}
				user, err = models.FindUserByGoogleID(db, id)
				if err != nil {
					slog.Error("Failed to find user", "error", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
					return
				}
			} else {
				slog.Error("Failed to register or login user", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}
		}
	case "h":
		if !config.Auth.GitHub.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "GitHub auth is disabled"})
			return
		}
		urldata := url.Values{}
		urldata.Set("code", data.Code)
		urldata.Set("client_id", config.Auth.GitHub.ClientID)
		urldata.Set("client_secret", config.Auth.GitHub.ClientSecret)

		tokenURL := fmt.Sprintf(
			"https://github.com/login/oauth/access_token?code=%s&client_id=%s&client_secret=%s",
			data.Code,
			config.Auth.GitHub.ClientID,
			config.Auth.GitHub.ClientSecret)

		resp, err := utils.HTTPRequest(c, http.MethodPost, tokenURL, strings.NewReader(urldata.Encode()), map[string]string{
			"Accept": "application/json",
		})
		if err != nil {
			slog.Error("Failed to make request", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slog.Error("Failed to get token", "status", resp.StatusCode)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		tokenResponse := v2.GitHubTokenResponse{}

		err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
		if err != nil {
			slog.Error("Failed to decode response", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		id, err := apis.GetGitHubUserID(c, tokenResponse.AccessToken)
		if err != nil {
			slog.Error("Failed to get GitHub user ID", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		user, err = models.FindUserByGitHubID(db, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) && config.Registration.Enabled {
				// Create user
				err = db.Create(&models.User{
					GitHubUserID: nulltype.NullInt64Of(int64(id)),
				}).Error
				if err != nil {
					slog.Error("Failed to create user", "error", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
					return
				}
				user, err = models.FindUserByGitHubID(db, id)
				if err != nil {
					slog.Error("Failed to find user", "error", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
					return
				}
			} else {
				slog.Error("Failed to register or login user", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}
		}
	case "c":
		if !config.Auth.Custom.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Custom auth is disabled"})
			return
		}
		urldata := url.Values{}
		urldata.Set("code", data.Code)
		urldata.Set("client_id", config.Auth.Custom.ClientID)
		urldata.Set("client_secret", config.Auth.Custom.ClientSecret)
		urldata.Set("grant_type", "authorization_code")
		urldata.Set("scope", "user:email")
		urldata.Set("redirect_uri", fmt.Sprintf(config.HTTP.BackendURL+"/v2/auth/c/redirect/"))

		resp, err := utils.HTTPRequest(c, http.MethodPost, config.Auth.Custom.TokenURL, strings.NewReader(urldata.Encode()), map[string]string{
			"Accept":       "application/json",
			"Content-Type": "application/x-www-form-urlencoded",
		})
		if err != nil {
			slog.Error("Failed to make request", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slog.Error("Failed to get token", "status", resp.StatusCode)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		tokenResponse := v2.GitHubTokenResponse{}

		err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
		if err != nil {
			slog.Error("Failed to decode response", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		id, err := apis.GetCustomUserID(c, config.Auth.Custom.UserURL, tokenResponse.AccessToken)
		if err != nil {
			slog.Error("Failed to get GitHub user ID", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		user, err = models.FindUserByCustomID(db, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) && config.Registration.Enabled {
				// Create user
				err = db.Create(&models.User{
					CustomUserID: nulltype.NullInt64Of(int64(id)),
				}).Error
				if err != nil {
					slog.Error("Failed to create user", "error", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
					return
				}
				user, err = models.FindUserByCustomID(db, id)
				if err != nil {
					slog.Error("Failed to find user", "error", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
					return
				}
			} else {
				slog.Error("Failed to register or login user", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is invalid"})
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
