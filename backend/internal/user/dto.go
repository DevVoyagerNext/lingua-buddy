package user

import "lingua-buddy/internal/models"

// View 是返回给前端的用户视图（不含密码哈希）。
type View struct {
	ID                 uint64  `json:"id"`
	Username           string  `json:"username"`
	Email              *string `json:"email"`
	RegistrationMethod string  `json:"registration_method"`
	EnglishLevel       string  `json:"english_level"`
}

// ToView 把模型转换为视图。
func ToView(u *models.User) View {
	return View{
		ID:                 u.ID,
		Username:           u.Username,
		Email:              u.Email,
		RegistrationMethod: u.RegistrationMethod,
		EnglishLevel:       u.EnglishLevel,
	}
}
