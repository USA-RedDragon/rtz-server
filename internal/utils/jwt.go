package utils

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserJWT struct {
	jwt.RegisteredClaims
	Identity uint `json:"identity"`
}

type DeviceJWT struct {
	jwt.RegisteredClaims
	Identity string `json:"identity"`
}

func (u UserJWT) GetAudience() (jwt.ClaimStrings, error) {
	return u.Audience, nil
}

func (u UserJWT) GetIssuer() (string, error) {
	return u.Issuer, nil
}

func (u UserJWT) GetSubject() (string, error) {
	return u.Subject, nil
}

func (u UserJWT) GetIssuedAt() (*jwt.NumericDate, error) {
	return u.IssuedAt, nil
}

func (u UserJWT) GetExpirationTime() (*jwt.NumericDate, error) {
	return u.ExpiresAt, nil
}

func (u UserJWT) GetNotBefore() (*jwt.NumericDate, error) {
	return u.NotBefore, nil
}

func GenerateJWT(signingKey string, userID uint) (string, error) {
	now := time.Now()
	claims := UserJWT{
		Identity: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(signingKey))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func VerifyJWT(signingKey string, tokenString string) (uint, error) {
	claims := new(UserJWT)
	token, err := jwt.NewParser(
		jwt.WithLeeway(5*time.Minute),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name})).
		ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("invalid signing method: %s", token.Header["alg"])
			}
			var ok bool
			claims, ok = token.Claims.(*UserJWT)
			if !ok {
				return nil, errors.New("invalid claims")
			}

			// ParseWithClaims will skip expiration check
			// if expiration has default value;
			// forcing a check and exiting if not set
			if claims.ExpiresAt == nil {
				return nil, errors.New("token has no expiration")
			}

			return []byte(signingKey), nil
		})
	if err != nil {
		return 0, err
	}
	if !token.Valid {
		return 0, errors.New("invalid token")
	}
	return claims.Identity, nil
}

func VerifyDeviceJWT(did string, signingKey string, tokenString string) error {
	claims := new(DeviceJWT)
	token, err := jwt.NewParser(
		jwt.WithLeeway(5*time.Minute),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name})).
		ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("invalid signing method: %s", token.Header["alg"])
			}
			var ok bool
			claims, ok = token.Claims.(*DeviceJWT)
			if !ok {
				return nil, errors.New("invalid claims")
			}

			// ParseWithClaims will skip expiration check
			// if expiration has default value;
			// forcing a check and exiting if not set
			if claims.ExpiresAt == nil {
				return nil, errors.New("token has no expiration")
			}

			if claims.Identity != did {
				return nil, errors.New("identity does not match device")
			}

			blk, _ := pem.Decode([]byte(signingKey))
			key, err := x509.ParsePKIXPublicKey(blk.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse public key: %w", err)
			}
			return key, nil
		})
	if err != nil {
		return err
	}
	if !token.Valid {
		return errors.New("invalid token")
	}
	return nil
}
