package v1

import "github.com/USA-RedDragon/rtz-server/internal/db/models"

type GETMeResponse struct {
	Email          string `json:"email"`
	ID             string `json:"id"`
	Prime          bool   `json:"prime"`
	RegisteredDate uint   `json:"regdate"`
	Superuser      bool   `json:"superuser"`
	UserID         string `json:"user_id"`
	Username       string `json:"username"`
}

type EligibleFeatures struct {
	Navigation bool `json:"nav"`
	Prime      bool `json:"prime"`
	PrimeData  bool `json:"prime_data"`
}

type GETMyDevicesResponse struct {
	models.Device
	Alias            string           `json:"alias"`
	AthenaHost       string           `json:"athena_host"`
	EligibleFeatures EligibleFeatures `json:"eligible_features"`
	IgnoreUploads    bool             `json:"ignore_uploads"`
	IsOwner          bool             `json:"is_owner"`
}
