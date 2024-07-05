package v2

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/db/models"
	v2 "github.com/USA-RedDragon/connect-server/internal/server/apimodels/v2"
	"github.com/USA-RedDragon/connect-server/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func POSTPilotPair(c *gin.Context) {
	var data v2.POSTPilotPairRequest

	data.PairToken = c.PostForm("pair_token")
	if data.PairToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pair_token is required"})
		return
	}

	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		slog.Error("Failed to get db from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	var claims = new(v2.PilotPairJWTClaims)

	token, err := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name})).
		ParseWithClaims(data.PairToken, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				slog.Error("Invalid signing method", "method", token.Header["alg"])
			}
			claims = token.Claims.(*v2.PilotPairJWTClaims)

			// ParseWithClaims will skip expiration check
			// if expiration has default value;
			// forcing a check and exiting if not set
			if claims.ExpiresAt == nil {
				return nil, errors.New("token has no expiration")
			}

			if !claims.Pair {
				return nil, errors.New("pair_token is not a pair token")
			}

			if claims.Identity == "" {
				return nil, errors.New("pair_token has no identity")
			}

			device, err := models.FindDeviceByDongleID(db, claims.Identity)
			if err != nil {
				return nil, errors.New("pair_token has invalid identity")
			}

			blk, _ := pem.Decode([]byte(device.PublicKey))
			key, err := x509.ParsePKIXPublicKey(blk.Bytes)
			if err != nil {
				slog.Error("Failed to parse public key", "error", err)
				return nil, errors.New("pair_token has invalid identity")
			}

			return key, nil
		})
	if err != nil {
		slog.Error("Failed to parse pair token", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "pair_token is invalid"})
		return
	}

	if !token.Valid {
		slog.Error("Invalid token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "pair_token is invalid"})
		return
	}

	device, err := models.FindDeviceByDongleID(db, claims.Identity)
	if err != nil {
		slog.Error("Failed to find device", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	user, ok := c.MustGet("user").(*models.User)
	if !ok {
		slog.Error("Failed to get user from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	firstPair := !device.IsPaired

	err = db.Model(&device).Update("owner_id", user.ID).Update("is_paired", true).Error
	if err != nil {
		slog.Error("Failed to pair device", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"first_pair": firstPair})
}

func POSTPilotAuth(c *gin.Context) {
	config, ok := c.MustGet("config").(*config.Config)
	if !ok {
		slog.Error("Failed to get config from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

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

	var claims = new(v2.RegisterJWTClaims)

	token, err := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name})).
		ParseWithClaims(param_register_token, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				slog.Error("Invalid signing method", "method", token.Header["alg"])
			}
			claims = token.Claims.(*v2.RegisterJWTClaims)

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
		slog.Error("Failed to parse register token", "error", err)
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
}