package server

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/USA-RedDragon/connect-server/internal/apis"
	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/db/models"
	"github.com/USA-RedDragon/connect-server/internal/events"
	websocketControllers "github.com/USA-RedDragon/connect-server/internal/server/websocket"
	"github.com/USA-RedDragon/connect-server/internal/utils"
	"github.com/USA-RedDragon/connect-server/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func applyRoutes(r *gin.Engine, config *config.Config, eventsChannel chan events.Event) {
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	authMiddleware := requireAuth(config)
	jwtAuthMiddleware := requireJWTAuth(config)

	apiV1 := r.Group("/v1")
	apiV1.GET("/me", jwtAuthMiddleware, func(c *gin.Context) {
		user, ok := c.MustGet("user").(*models.User)
		if !ok {
			slog.Error("Failed to get user from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		type userResponse struct {
			Email          string `json:"email"`
			ID             string `json:"id"`
			Prime          bool   `json:"prime"`
			RegisteredDate uint   `json:"regdate"`
			Superuser      bool   `json:"superuser"`
			UserID         string `json:"user_id"`
			Username       string `json:"username"`
		}
		userResp := userResponse{
			Email:          "no emails here",
			ID:             fmt.Sprintf("%d", user.ID),
			Prime:          true,
			RegisteredDate: uint(user.CreatedAt.Unix()),
			Superuser:      user.Superuser,
		}
		if user.GitHubUserID != 0 {
			userResp.UserID = fmt.Sprintf("%d", user.GitHubUserID)
		} else {
			userResp.UserID = user.GoogleUserID
		}

		c.JSON(http.StatusOK, userResp)
	})

	apiV1.GET("/navigation/:dongle_id/next", setDevice(), authMiddleware, func(c *gin.Context) {
		slog.Info("Get Next Navigation", "url", c.Request.URL.String())
	})

	apiV1.DELETE("/navigation/:dongle_id/next", setDevice(), authMiddleware, func(c *gin.Context) {
		slog.Info("Delete Next Navigation", "url", c.Request.URL.String())
	})

	apiV1.GET("/navigation/:dongle_id/locations", setDevice(), authMiddleware, func(c *gin.Context) {
		slog.Info("Get Locations", "url", c.Request.URL.String())
	})

	apiV11 := r.Group("/v1.1")
	apiV11.GET("/devices/:dongle_id/", setDevice(), authMiddleware, func(c *gin.Context) {
		db, ok := c.MustGet("db").(*gorm.DB)
		if !ok {
			slog.Error("Failed to get db from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		dongleID := c.Param("dongle_id")
		if dongleID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dongle_id is required"})
			return
		}

		device, err := models.FindDeviceByDongleID(db, dongleID)
		if err != nil {
			slog.Error("Failed to find device", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		c.JSON(http.StatusOK, device)
	})

	apiV11.GET("/devices/:dongle_id/stats", setDevice(), authMiddleware, func(c *gin.Context) {
		slog.Info("Get Stats", "url", c.Request.URL.String())
	})

	apiV14 := r.Group("/v1.4")
	apiV14.GET("/:dongle_id/upload_url", setDevice(), authMiddleware, func(c *gin.Context) {
		slog.Info("Get Upload URL", "url", c.Request.URL.String())
	})

	apiV2 := r.Group("/v2")

	apiV2.POST("/auth", func(c *gin.Context) {
		var data struct {
			Provider string
			Code     string
		}

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

			var tokenResponse struct {
				AccessToken  string `json:"access_token"`
				TokenType    string `json:"token_type"`
				ExpiresIn    int    `json:"expires_in"`
				Scope        string `json:"scope"`
				RefreshToken string `json:"refresh_token"`
			}

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

			var tokenResponse struct {
				AccessToken string `json:"access_token"`
			}

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
	})

	// Google Auth Redirect
	apiV2.GET("/auth/g/redirect", func(c *gin.Context) {
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
	})

	// GitHub Auth Redirect
	apiV2.GET("/auth/h/redirect", func(c *gin.Context) {
		code := c.Query("code")
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
			return
		}

		// Redirect to the app with the code
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/auth/?provider=h&code=%s", config.HTTP.FrontendURL, code))
	})

	apiV2.POST("/pilotpair", func(c *gin.Context) {
	})

	apiV2.POST("/pilotauth", func(c *gin.Context) {
		if !config.Registration.Enabled {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registration is disabled"})
			return
		}
		param_imei, ok := c.GetQuery("imei")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei is required"})
			return
		}
		if len(param_imei) != 15 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei must be 15 characters"})
			return
		}
		imei, err := strconv.ParseInt(param_imei, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei is not an integer"})
		}
		if !utils.LuhnValid(int(imei)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei is invalid"})
			return
		}

		param_imei2, ok := c.GetQuery("imei2")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei2 is required"})
			return
		}
		var imei2 int64
		if len(param_imei2) != 0 {
			imei2, err = strconv.ParseInt(param_imei2, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "imei2 is not an integer"})
			}
		}
		if !utils.LuhnValid(int(imei2)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei2 is invalid"})
			return
		}

		param_serial, ok := c.GetQuery("serial")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "serial is required"})
			return
		}
		if len(param_serial) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "serial is required"})
			return
		}

		param_public_key, ok := c.GetQuery("public_key")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "public_key is required"})
			return
		}
		if len(param_public_key) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "public_key is required"})
			return
		}

		param_register_token, ok := c.GetQuery("register_token")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "register_token is required"})
			return
		}

		blk, _ := pem.Decode([]byte(param_public_key))
		key, err := x509.ParsePKIXPublicKey(blk.Bytes)
		if err != nil {
			slog.Error("Failed to parse public key", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "public_key is invalid"})
			return
		}

		type Claims struct {
			Register bool `json:"register,omitempty"`
			jwt.RegisteredClaims
		}

		var claims *Claims

		token, err := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name})).
			ParseWithClaims(param_register_token, new(Claims), func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					slog.Error("Invalid signing method", "method", token.Header["alg"])
				}
				claims = token.Claims.(*Claims)

				// ParseWithClaims will skip expiration check
				// if expiration has default value;
				// forcing a check and exiting if not set
				if claims.ExpiresAt == nil {
					return nil, errors.New("token has no expiration")
				}

				if !claims.Register {
					return nil, errors.New("register_token is not a register token")
				}

				return key, nil
			})
		if err != nil {
			slog.Error("Failed to parse token", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "register_token is invalid"})
			return
		}

		if !token.Valid {
			slog.Error("Invalid token")
			c.JSON(http.StatusBadRequest, gin.H{"error": "register_token is invalid"})
			return
		}

		db, ok := c.MustGet("db").(*gorm.DB)
		if !ok {
			slog.Error("Failed to get db from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		_, err = models.FindDeviceBySerial(db, param_serial)
		// We can ignore the error here, as we're just checking if the device exists
		if err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "serial is already registered"})
			return
		}

		dongleID, err := models.GenerateDongleID(db)
		if err != nil {
			slog.Error("Failed to generate dongle ID", "error", err)
		}

		err = db.Create(&models.Device{
			DongleID:  dongleID,
			Serial:    param_serial,
			PublicKey: param_public_key,
		}).Error
		if err != nil {
			slog.Error("Failed to create device", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"dongle_id": dongleID})
	})

	r.NoRoute(func(c *gin.Context) {
		slog.Warn("Not Found", "path", c.Request.URL.Path)
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	})

	wsV2 := r.Group("/ws/v2")
	wsV2.GET("/:dongle_id", setDevice(), requireCookieAuth(config), websocket.CreateHandler(websocketControllers.CreateEventsWebsocket(eventsChannel), config))
}
