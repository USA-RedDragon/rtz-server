package v1dot4

type UploadURLResponse struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}
