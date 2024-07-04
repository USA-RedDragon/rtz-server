package v2

import "github.com/golang-jwt/jwt/v5"

type RegisterJWTClaims struct {
	Register bool `json:"register,omitempty"`
	jwt.RegisteredClaims
}
