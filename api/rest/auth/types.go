package auth

import "codeberg.org/algorave/server/algorave/users"

// AuthResponse returned after successful OAuth callback
type AuthResponse struct {
	User  *users.User `json:"user"`
	Token string      `json:"token"`
}

// UserResponse wraps user data
type UserResponse struct {
	User *users.User `json:"user"`
}

// MessageResponse for simple success messages
type MessageResponse struct {
	Message string `json:"message"`
}

// UpdateProfileRequest for updating user profile
type UpdateProfileRequest struct {
	Name      string `json:"name" binding:"required,max=100"`
	AvatarURL string `json:"avatar_url" binding:"max=500"`
}
