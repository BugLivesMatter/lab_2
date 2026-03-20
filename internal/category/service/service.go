package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/lab2/rest-api/internal/category/domain"
	"github.com/lab2/rest-api/internal/category/dto"
	categoryrepo "github.com/lab2/rest-api/internal/category/repository"
	productrepo "github.com/lab2/rest-api/internal/product/repository"
	"github.com/lab2/rest-api/pkg/apperror"
	"github.com/lab2/rest-api/pkg/pagination"
)

type CategoryService interface {
	Create(ctx context.Context, req *dto.CreateCategoryRequest) (*domain.Category, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	List(ctx context.Context, page, limit int) ([]domain.Category, int64, int, error)
	Update(ctx context.Context, id uuid.UUID, req *dto.UpdateCategoryRequest) (*domain.Category, error)
	Patch(ctx context.Context, id uuid.UUID, req *dto.PatchCategoryRequest) (*domain.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type categoryService struct {
	repo        categoryrepo.CategoryRepository
	productRepo productrepo.ProductRepository
}

func NewCategoryService(repo categoryrepo.CategoryRepository, productRepo productrepo.ProductRepository) CategoryService {
	return &categoryService{repo: repo, productRepo: productRepo}
}

func (s *categoryService) Create(ctx context.Context, req *dto.CreateCategoryRequest) (*domain.Category, error) {
	status := req.Status
	if status == "" {
		status = "active"
	}
	category := &domain.Category{
		Name:        req.Name,
		Description: req.Description,
		Status:      status,
	}
	if err := s.repo.Create(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *categoryService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, err
	}
	return category, nil
}

func (s *categoryService) List(ctx context.Context, page, limit int) ([]domain.Category, int64, int, error) {
	if page < 1 {
		page = pagination.DefaultPage
	}
	if limit < 1 {
		limit = pagination.DefaultLimit
	}
	if limit > pagination.MaxLimit {
		limit = pagination.MaxLimit
	}
	offset := (page - 1) * limit
	categories, total, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, 0, err
	}
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	return categories, total, totalPages, nil
}

func (s *categoryService) Update(ctx context.Context, id uuid.UUID, req *dto.UpdateCategoryRequest) (*domain.Category, error) {
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, err
	}
	category.Name = req.Name
	category.Description = req.Description
	category.Status = req.Status
	if err := s.repo.Update(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *categoryService) Patch(ctx context.Context, id uuid.UUID, req *dto.PatchCategoryRequest) (*domain.Category, error) {
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, err
	}
	if req.Name != nil {
		category.Name = *req.Name
	}
	if req.Description != nil {
		category.Description = *req.Description
	}
	if req.Status != nil {
		category.Status = *req.Status
	}
	if err := s.repo.Update(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *categoryService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrNotFound
		}
		return err
	}
	count, err := s.productRepo.CountByCategoryID(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return apperror.ErrConflict
	}
	return s.repo.Delete(ctx, id)
}
