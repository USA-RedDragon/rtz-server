package v1dot4

type UploadURLResponse struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type RouteInfo struct {
	Motonic string
	Route   string
	Segment string
}
