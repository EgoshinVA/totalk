package domain

// UserResponse — публичное представление пользователя (без хэша пароля).
// Используется в /me, /login, /register/finalize.
type UserResponse struct {
	ID         int64   `json:"id"`
	Email      string  `json:"email"`
	Name       string  `json:"name"`
	SurName    string  `json:"surName"`
	Patronymic *string `json:"patronymic,omitempty"`
	AvatarURL  *string `json:"avatarUrl,omitempty"`
}

// TokenPair — пара токенов, возвращаемая при логине и финализации регистрации.
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// AuthResponse объединяет токены и данные юзера.
type AuthResponse struct {
	TokenPair
	User UserResponse `json:"user"`
}

// UserFromModel конвертирует domain.User в UserResponse.
func UserFromModel(u *User) UserResponse {
	return UserResponse{
		ID:         u.ID,
		Email:      u.Email,
		Name:       u.Name,
		SurName:    u.SurName,
		Patronymic: u.Patronymic,
		AvatarURL:  u.AvatarURL,
	}
}
