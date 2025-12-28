package health

type Response struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version,omitempty"`
}

type PingResponse struct {
	Message string `json:"message"`
}
