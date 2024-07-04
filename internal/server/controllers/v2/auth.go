package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/USA-RedDragon/connect-server/internal/apis"
	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/db/models"
	v2 "github.com/USA-RedDragon/connect-server/internal/server/apimodels/v2"
	"github.com/USA-RedDragon/connect-server/internal/utils"
	"github.com/gin-gonic/gin"
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
		tokenURL := "https://oauth2.googleapis.com/token"

		urldata := url.Values{}
		urldata.Set("code", data.Code)
		urldata.Set("client_id", config.Auth.Google.ClientID)
		urldata.Set("client_secret", config.Auth.Google.ClientSecret)
		urldata.Set("redirect_uri", config.Auth.Google.RedirectURI)
		urldata.Set("grant_type", "authorization_code")

		resp, err := utils.HTTPRequest(http.MethodPost, tokenURL, strings.NewReader(urldata.Encode()), map[string]string{
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

		id, err := apis.GetGoogleUserID(tokenResponse.AccessToken)
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
					GoogleUserID: id,
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
		urldata := url.Values{}
		urldata.Set("code", data.Code)
		urldata.Set("client_id", config.Auth.GitHub.ClientID)
		urldata.Set("client_secret", config.Auth.GitHub.ClientSecret)

		tokenURL := fmt.Sprintf(
			"https://github.com/login/oauth/access_token?code=%s&client_id=%s&client_secret=%s",
			data.Code,
			config.Auth.GitHub.ClientID,
			config.Auth.GitHub.ClientSecret)

		resp, err := utils.HTTPRequest(http.MethodPost, tokenURL, strings.NewReader(urldata.Encode()), map[string]string{
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

		id, err := apis.GetGitHubUserID(tokenResponse.AccessToken)
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
					GitHubUserID: id,
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

func GETGoogleRedirect(c *gin.Context) {
	config, ok := c.MustGet("config").(*config.Config)
	if !ok {
		slog.Error("Failed to get config from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	error := c.Query("error")
	if error != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": error})
		return
	}
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
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

	// Redirect to the app with the code
	c.Redirect(http.StatusFound, fmt.Sprintf("%s/auth/?provider=g&code=%s", config.HTTP.FrontendURL, code))
}

func GETGitHubRedirect(c *gin.Context) {
	config, ok := c.MustGet("config").(*config.Config)
	if !ok {
		slog.Error("Failed to get config from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	// Redirect to the app with the code
	c.Redirect(http.StatusFound, fmt.Sprintf("%s/auth/?provider=h&code=%s", config.HTTP.FrontendURL, code))
}
