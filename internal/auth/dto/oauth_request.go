package dto

// OAuthCallbackRequest содержит данные от OAuth провайдера
type OAuthCallbackRequest struct {
	Code  string `form:"code"`
	State string `form:"state"`
}

// OAuthUserInfo содержит информацию о пользователе от провайдера
type OAuthUserInfo struct {
	ID        string `json:"id"`
	Email     string `json:"default_email"`
	Login     string `json:"login"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
