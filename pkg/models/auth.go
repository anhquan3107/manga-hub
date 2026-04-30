package models

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=72"`
	Email    string `json:"email" binding:"required,email"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,min=6,max=72"`
	NewPassword     string `json:"new_password" binding:"required,min=6,max=72"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
