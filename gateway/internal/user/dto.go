package user

type LoginRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	FullName string `json:"full_name"`
	Email string `json:"email"`
	Password string `json:"password"`
}

type Dto struct {
	FullName string `json:"full_name"`
	Email string `json:"email"`
}

func ToUser(r *RegisterRequest) *User {
	return &User{
		FullName: r.FullName,
		Email: r.Email,
	}
}