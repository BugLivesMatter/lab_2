package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/lab2/rest-api/internal/product/dto"
	"github.com/lab2/rest-api/internal/product/service"
	"github.com/lab2/rest-api/pkg/apperror"
	"github.com/lab2/rest-api/pkg/pagination"
)

type ProductHandler struct {
	svc service.ProductService
}

func NewProductHandler(svc service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

// Create обрабатывает POST /products
// @Summary Создать продукт
// @Tags products
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body dto.CreateProductRequest true "Тело запроса"
// @Success 201 {object} dto.ProductResponse "продукт создан"
// @Failure 400 {object} apperror.ErrorResponse
// @Failure 401 {object} apperror.ErrorResponse
// @Failure 403 {object} apperror.ErrorResponse
// @Failure 404 {object} apperror.ErrorResponse
// @Failure 500 {object} apperror.ErrorResponse
// @Router /products [post]
func (h *ProductHandler) Create(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		status := apperror.StatusFromError(err)
		c.JSON(status, gin.H{"error": apperror.Message(err, status)})
		return
	}
	c.JSON(http.StatusCreated, dto.ProductToResponse(product))
}

// GetByID обрабатывает GET /products/{id}
// @Summary Получить продукт по UUID
// @Tags products
// @Produce json
// @Security CookieAuth
// @Param id path string true "UUID продукта"
// @Success 200 {object} dto.ProductResponse "продукт найден"
// @Failure 400 {object} apperror.ErrorResponse
// @Failure 401 {object} apperror.ErrorResponse
// @Failure 403 {object} apperror.ErrorResponse
// @Failure 404 {object} apperror.ErrorResponse
// @Failure 500 {object} apperror.ErrorResponse
// @Router /products/{id} [get]
func (h *ProductHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	product, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		status := apperror.StatusFromError(err)
		c.JSON(status, gin.H{"error": apperror.Message(err, status)})
		return
	}
	c.JSON(http.StatusOK, dto.ProductToResponse(product))
}

// List обрабатывает GET /products
// @Summary Список продуктов
// @Tags products
// @Produce json
// @Security CookieAuth
// @Param category_id query string false "Фильтр по UUID категории"
// @Param page query int false "Номер страницы" example(1)
// @Param limit query int false "Количество элементов на странице" example(10)
// @Success 200 {object} dto.ProductListResponse "список продуктов"
// @Failure 400 {object} apperror.ErrorResponse
// @Failure 401 {object} apperror.ErrorResponse
// @Failure 403 {object} apperror.ErrorResponse
// @Failure 500 {object} apperror.ErrorResponse
// @Router /products [get]
func (h *ProductHandler) List(c *gin.Context) {
	page := pagination.DefaultPage
	limit := pagination.DefaultLimit
	var categoryID *uuid.UUID
	if cid := c.Query("category_id"); cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			categoryID = &id
		}
	}
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v >= 1 && v <= pagination.MaxLimit {
			limit = v
		}
	}
	products, total, totalPages, err := h.svc.List(c.Request.Context(), page, limit, categoryID)
	if err != nil {
		status := apperror.StatusFromError(err)
		c.JSON(status, gin.H{"error": apperror.Message(err, status)})
		return
	}
	data := make([]dto.ProductResponse, len(products))
	for i := range products {
		data[i] = dto.ProductToResponse(&products[i])
	}
	c.JSON(http.StatusOK, dto.ProductListResponse{
		Data: data,
		Meta: pagination.Meta{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	})
}

// Update обрабатывает PUT /products/{id}
// @Summary Полное обновление продукта
// @Tags products
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "UUID продукта"
// @Param request body dto.UpdateProductRequest true "Тело запроса"
// @Success 200 {object} dto.ProductResponse "продукт обновлён"
// @Failure 400 {object} apperror.ErrorResponse
// @Failure 401 {object} apperror.ErrorResponse
// @Failure 403 {object} apperror.ErrorResponse
// @Failure 404 {object} apperror.ErrorResponse
// @Failure 500 {object} apperror.ErrorResponse
// @Router /products/{id} [put]
func (h *ProductHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		status := apperror.StatusFromError(err)
		c.JSON(status, gin.H{"error": apperror.Message(err, status)})
		return
	}
	c.JSON(http.StatusOK, dto.ProductToResponse(product))
}

// Patch обрабатывает PATCH /products/{id}
// @Summary Частичное обновление продукта
// @Tags products
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "UUID продукта"
// @Param request body dto.PatchProductRequest true "Тело запроса"
// @Success 200 {object} dto.ProductResponse "продукт обновлён"
// @Failure 400 {object} apperror.ErrorResponse
// @Failure 401 {object} apperror.ErrorResponse
// @Failure 403 {object} apperror.ErrorResponse
// @Failure 404 {object} apperror.ErrorResponse
// @Failure 500 {object} apperror.ErrorResponse
// @Router /products/{id} [patch]
func (h *ProductHandler) Patch(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req dto.PatchProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.svc.Patch(c.Request.Context(), id, &req)
	if err != nil {
		status := apperror.StatusFromError(err)
		c.JSON(status, gin.H{"error": apperror.Message(err, status)})
		return
	}
	c.JSON(http.StatusOK, dto.ProductToResponse(product))
}

// Delete обрабатывает DELETE /products/{id}
// @Summary Удалить продукт (soft delete)
// @Tags products
// @Produce json
// @Security CookieAuth
// @Param id path string true "UUID продукта"
// @Success 204 "продукт удалён"
// @Failure 400 {object} apperror.ErrorResponse
// @Failure 401 {object} apperror.ErrorResponse
// @Failure 403 {object} apperror.ErrorResponse
// @Failure 404 {object} apperror.ErrorResponse
// @Failure 500 {object} apperror.ErrorResponse
// @Router /products/{id} [delete]
func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		status := apperror.StatusFromError(err)
		c.JSON(status, gin.H{"error": apperror.Message(err, status)})
		return
	}
	c.Status(http.StatusNoContent)
}
