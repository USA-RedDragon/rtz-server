package v2

import "github.com/golang-jwt/jwt/v5"

type RegisterJWTClaims struct {
	Register bool `json:"register,omitempty"`
	jwt.RegisteredClaims
}

type POSTPilotPairRequest struct {
	PairToken string `json:"pair_token"`
}

type PilotPairJWTClaims struct {
	Identity string `json:"identity"`
	Pair     bool   `json:"pair,omitempty"`
	jwt.RegisteredClaims
}
