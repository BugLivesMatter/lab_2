package handler

import (
	"errors"
	"net/http"

	"github.com/lab2/rest-api/internal/service"
)

func statusFromError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, service.ErrBadRequest) {
		return http.StatusBadRequest
	}
	if errors.Is(err, service.ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, service.ErrConflict) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

func errorMessage(err error, status int) string {
	if errors.Is(err, service.ErrBadRequest) {
		return "invalid category id"
	}
	if errors.Is(err, service.ErrNotFound) {
		return "not found"
	}
	if errors.Is(err, service.ErrConflict) {
		return "conflict"
	}
	if status == http.StatusInternalServerError {
		return "internal server error"
	}
	return err.Error()
}
