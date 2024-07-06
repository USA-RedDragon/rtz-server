package v1dot1

type StatsResponse struct {
	All  Stats `json:"all"`
	Week Stats `json:"week"`
}

type Stats struct {
	Distance float64 `json:"distance"`
	Minutes  int64   `json:"minutes"`
	Routes   int64   `json:"routes"`
}
