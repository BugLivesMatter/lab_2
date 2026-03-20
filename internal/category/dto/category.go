package dto

import (
	"github.com/lab2/rest-api/internal/category/domain"
	"github.com/lab2/rest-api/pkg/pagination"
)

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required" example:"Электроника"`
	Description string `json:"description" example:"Товары и устройства"`
	Status      string `json:"status" binding:"omitempty,oneof=active hidden" example:"active"`
}

type UpdateCategoryRequest struct {
	Name        string `json:"name" binding:"required" example:"Электроника"`
	Description string `json:"description" example:"Товары и устройства"`
	Status      string `json:"status" binding:"required,oneof=active hidden" example:"active"`
}

type PatchCategoryRequest struct {
	Name        *string `json:"name" example:"Электроника"`
	Description *string `json:"description" example:"Товары и устройства"`
	Status      *string `json:"status" binding:"omitempty,oneof=active hidden" example:"hidden"`
}

type CategoryResponse struct {
	ID          string `json:"id" format:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string `json:"name" example:"Электроника"`
	Description string `json:"description" example:"Товары и устройства"`
	Status      string `json:"status" example:"active"`
	CreatedAt   string `json:"createdAt" format:"date-time" example:"2026-03-19T13:18:48.000Z"`
}

type CategoryListResponse struct {
	Data []CategoryResponse `json:"data"`
	Meta pagination.Meta    `json:"meta"`
}

func CategoryToResponse(c *domain.Category) CategoryResponse {
	return CategoryResponse{
		ID:          c.ID.String(),
		Name:        c.Name,
		Description: c.Description,
		Status:      c.Status,
		CreatedAt:   c.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
}
