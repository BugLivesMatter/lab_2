package dto

import (
	"github.com/lab2/rest-api/internal/product/domain"
	"github.com/lab2/rest-api/pkg/pagination"
)

type CreateProductRequest struct {
	CategoryID  string  `json:"categoryId" binding:"required" format:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string  `json:"name" binding:"required" example:"Ноутбук"`
	Description string  `json:"description" example:"14 дюймов, 16GB RAM"`
	Price       float64 `json:"price" binding:"required,gte=0" example:"79990.50"`
	Status      string  `json:"status" binding:"omitempty,oneof=available out_of_stock discontinued" example:"available"`
}

type UpdateProductRequest struct {
	CategoryID  string  `json:"categoryId" binding:"required" format:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string  `json:"name" binding:"required" example:"Ноутбук"`
	Description string  `json:"description" example:"14 дюймов, 16GB RAM"`
	Price       float64 `json:"price" binding:"required,gte=0" example:"79990.50"`
	Status      string  `json:"status" binding:"required,oneof=available out_of_stock discontinued" example:"available"`
}

type PatchProductRequest struct {
	CategoryID  *string  `json:"categoryId" format:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        *string  `json:"name" example:"Ноутбук"`
	Description *string  `json:"description" example:"14 дюймов, 16GB RAM"`
	Price       *float64 `json:"price" binding:"omitempty,gte=0" example:"79990.50"`
	Status      *string  `json:"status" binding:"omitempty,oneof=available out_of_stock discontinued" example:"out_of_stock"`
}

type ProductResponse struct {
	ID           string  `json:"id" format:"uuid" example:"550e8400-e29b-41d4-a716-446655440001"`
	CategoryID   string  `json:"categoryId" format:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	CategoryName string  `json:"categoryName,omitempty" example:"Электроника"`
	Name         string  `json:"name" example:"Ноутбук"`
	Description  string  `json:"description" example:"14 дюймов, 16GB RAM"`
	Price        float64 `json:"price" example:"79990.50"`
	Status       string  `json:"status" example:"available"`
	CreatedAt    string  `json:"createdAt" format:"date-time" example:"2026-03-19T13:18:48.000Z"`
}

type ProductListResponse struct {
	Data []ProductResponse `json:"data"`
	Meta pagination.Meta   `json:"meta"`
}

func ProductToResponse(p *domain.Product) ProductResponse {
	resp := ProductResponse{
		ID:          p.ID.String(),
		CategoryID:  p.CategoryID.String(),
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	if p.Category != nil {
		resp.CategoryName = p.Category.Name
	}
	return resp
}
