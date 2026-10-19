package http

// HTTP-specific DTOs for TargetSystem
type HTTPTargetSystemRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type HTTPTargetSystemResponse struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message,omitempty"`
}
