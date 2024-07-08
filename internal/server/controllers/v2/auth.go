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
		tokenURL := "https://oauth2.googleapis.com/token"

		urldata := url.Values{}
		urldata.Set("code", data.Code)
		urldata.Set("client_id", config.Auth.Google.ClientID)
		urldata.Set("client_secret", config.Auth.Google.ClientSecret)
		urldata.Set("redirect_uri", config.HTTP.BackendURL+"/v2/auth/g/redirect/")
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

	refererStr := c.Request.Header.Get("Referer")
	if refererStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Referer header is required"})
		return
	}

	referer, err := url.Parse(refererStr)
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
	queries.Add("provider", "g")

	authRedirect.RawQuery = queries.Encode()

	slog.Info("Google redirect", "Referer", c.Request.Header.Get("Referer"), "authRedirect", authRedirect.String())

	// Redirect to the app with the code
	c.Redirect(http.StatusFound, authRedirect.String())
}

func GETGitHubRedirect(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	refererStr := c.Request.Header.Get("Referer")
	if refererStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Referer header is required"})
		return
	}

	referer, err := url.Parse(refererStr)
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
	queries.Add("provider", "h")

	authRedirect.RawQuery = queries.Encode()

	slog.Info("GitHub redirect", "Referer", c.Request.Header.Get("Referer"), "authRedirect", authRedirect.String())

	// Redirect to the app with the code
	c.Redirect(http.StatusFound, authRedirect.String())
}
