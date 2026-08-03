package application

// @Description Health response
type HealthResponse struct {
	// @Description Health status
	Status string `json:"status" example:"ok"`
}
