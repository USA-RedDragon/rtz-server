package v1

type GETMeResponse struct {
	Email          string `json:"email"`
	ID             string `json:"id"`
	Prime          bool   `json:"prime"`
	RegisteredDate uint   `json:"regdate"`
	Superuser      bool   `json:"superuser"`
	UserID         string `json:"user_id"`
	Username       string `json:"username"`
}

type GETMyDevicesResponse struct {
}
