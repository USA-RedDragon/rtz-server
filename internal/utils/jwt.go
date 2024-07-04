package utils

import "github.com/golang-jwt/jwt/v5"

func GenerateJWT(signingKey string, userID uint, superuser bool) (string, error) {
	claims := jwt.MapClaims{
		"sub":       userID,
		"superuser": superuser,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(signingKey))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}
