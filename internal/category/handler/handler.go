package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/lab2/rest-api/internal/category/dto"
	"github.com/lab2/rest-api/internal/category/service"
	"github.com/lab2/rest-api/pkg/apperror"
	"github.com/lab2/rest-api/pkg/pagination"
)

type CategoryHandler struct {
	svc service.CategoryService
}

func NewCategoryHandler(svc service.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

// Create обрабатывает POST /categories
// @Summary Создать категорию
// @Tags categories
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body dto.CreateCategoryRequest true "Тело запроса"
// @Success 201 {object} dto.CategoryResponse "категория создана"
// @Failure 400 {object} apperror.ErrorResponse
// @Failure 401 {object} apperror.ErrorResponse
// @Failure 403 {object} apperror.ErrorResponse
// @Failure 404 {object} apperror.ErrorResponse
// @Failure 500 {object} apperror.ErrorResponse
// @Router /categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	var req dto.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	category, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		status := apperror.StatusFromError(err)
		c.JSON(status, gin.H{"error": apperror.Message(err, status)})
		return
	}
	c.JSON(http.StatusCreated, dto.CategoryToResponse(category))
}

// GetByID обрабатывает GET /categories/{id}
// @Summary Получить категорию по UUID
// @Tags categories
// @Produce json
// @Security CookieAuth
// @Param id path string true "UUID категории"
// @Success 200 {object} dto.CategoryResponse "категория найдена"
// @Failure 400 {object} apperror.ErrorResponse
// @Failure 401 {object} apperror.ErrorResponse
// @Failure 403 {object} apperror.ErrorResponse
// @Failure 404 {object} apperror.ErrorResponse
// @Failure 500 {object} apperror.ErrorResponse
// @Router /categories/{id} [get]
func (h *CategoryHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	category, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		status := apperror.StatusFromError(err)
		c.JSON(status, gin.H{"error": apperror.Message(err, status)})
		return
	}
	c.JSON(http.StatusOK, dto.CategoryToResponse(category))
}

// List обрабатывает GET /categories
// @Summary Список категорий
// @Tags categories
// @Produce json
// @Security CookieAuth
// @Param page query int false "Номер страницы" example(1)
// @Param limit query int false "Количество элементов на странице" example(10)
// @Success 200 {object} dto.CategoryListResponse "список категорий"
// @Failure 400 {object} apperror.ErrorResponse
// @Failure 401 {object} apperror.ErrorResponse
// @Failure 403 {object} apperror.ErrorResponse
// @Failure 500 {object} apperror.ErrorResponse
// @Router /categories [get]
func (h *CategoryHandler) List(c *gin.Context) {
	page := pagination.DefaultPage
	limit := pagination.DefaultLimit
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
	categories, total, totalPages, err := h.svc.List(c.Request.Context(), page, limit)
	if err != nil {
		status := apperror.StatusFromError(err)
		c.JSON(status, gin.H{"error": apperror.Message(err, status)})
		return
	}
	data := make([]dto.CategoryResponse, len(categories))
	for i := range categories {
		data[i] = dto.CategoryToResponse(&categories[i])
	}
	c.JSON(http.StatusOK, dto.CategoryListResponse{
		Data: data,
		Meta: pagination.Meta{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	})
}

// Update обрабатывает PUT /categories/{id}
// @Summary Полное обновление категории
// @Tags categories
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "UUID категории"
// @Param request body dto.UpdateCategoryRequest true "Тело запроса"
// @Success 200 {object} dto.CategoryResponse "категория обновлена"
// @Failure 400 {object} apperror.ErrorResponse
// @Failure 401 {object} apperror.ErrorResponse
// @Failure 403 {object} apperror.ErrorResponse
// @Failure 404 {object} apperror.ErrorResponse
// @Failure 500 {object} apperror.ErrorResponse
// @Router /categories/{id} [put]
func (h *CategoryHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req dto.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	category, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		status := apperror.StatusFromError(err)
		c.JSON(status, gin.H{"error": apperror.Message(err, status)})
		return
	}
	c.JSON(http.StatusOK, dto.CategoryToResponse(category))
}

// Patch обрабатывает PATCH /categories/{id}
// @Summary Частичное обновление категории
// @Tags categories
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param id path string true "UUID категории"
// @Param request body dto.PatchCategoryRequest true "Тело запроса"
// @Success 200 {object} dto.CategoryResponse "категория обновлена"
// @Failure 400 {object} apperror.ErrorResponse
// @Failure 401 {object} apperror.ErrorResponse
// @Failure 403 {object} apperror.ErrorResponse
// @Failure 404 {object} apperror.ErrorResponse
// @Failure 500 {object} apperror.ErrorResponse
// @Router /categories/{id} [patch]
func (h *CategoryHandler) Patch(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req dto.PatchCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	category, err := h.svc.Patch(c.Request.Context(), id, &req)
	if err != nil {
		status := apperror.StatusFromError(err)
		c.JSON(status, gin.H{"error": apperror.Message(err, status)})
		return
	}
	c.JSON(http.StatusOK, dto.CategoryToResponse(category))
}

// Delete обрабатывает DELETE /categories/{id}
// @Summary Удалить категорию (soft delete)
// @Tags categories
// @Produce json
// @Security CookieAuth
// @Param id path string true "UUID категории"
// @Success 204 "категория удалена"
// @Failure 400 {object} apperror.ErrorResponse
// @Failure 401 {object} apperror.ErrorResponse
// @Failure 403 {object} apperror.ErrorResponse
// @Failure 404 {object} apperror.ErrorResponse
// @Failure 409 {object} apperror.ErrorResponse
// @Failure 500 {object} apperror.ErrorResponse
// @Router /categories/{id} [delete]
func (h *CategoryHandler) Delete(c *gin.Context) {
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
