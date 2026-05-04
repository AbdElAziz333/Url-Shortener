package user

type LoginRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type Dto struct {
	Email string `json:"email"`
}