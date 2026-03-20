package apperror

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound   = errors.New("resource not found")
	ErrConflict   = errors.New("conflict")
	ErrBadRequest = errors.New("invalid request")
)

func StatusFromError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, ErrBadRequest) {
		return http.StatusBadRequest
	}
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrConflict) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

func Message(err error, status int) string {
	if errors.Is(err, ErrBadRequest) {
		return "invalid id"
	}
	if errors.Is(err, ErrNotFound) {
		return "not found"
	}
	if errors.Is(err, ErrConflict) {
		return "conflict"
	}
	if status == http.StatusInternalServerError {
		return "internal server error"
	}
	return err.Error()
}

// ErrorResponse описывает единый формат ошибки для Swagger-аннотаций.
type ErrorResponse struct {
	Error string `json:"error" example:"not found"`
}
