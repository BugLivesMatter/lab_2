package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/lab2/rest-api/internal/product/domain"
	"gorm.io/gorm"
)

type ProductRepository interface {
	Create(ctx context.Context, product *domain.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	List(ctx context.Context, offset, limit int, categoryID *uuid.UUID) ([]domain.Product, int64, error)
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByCategoryID(ctx context.Context, categoryID uuid.UUID) (int64, error)
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, product *domain.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *productRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	var product domain.Product
	err := r.db.WithContext(ctx).
		Preload("Category").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) List(ctx context.Context, offset, limit int, categoryID *uuid.UUID) ([]domain.Product, int64, error) {
	var products []domain.Product
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Product{}).Where("deleted_at IS NULL")
	if categoryID != nil {
		query = query.Where("category_id = ?", *categoryID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := r.db.WithContext(ctx).
		Preload("Category").
		Where("deleted_at IS NULL").
		Offset(offset).
		Limit(limit)
	if categoryID != nil {
		findQuery = findQuery.Where("category_id = ?", *categoryID)
	}
	if err := findQuery.Find(&products).Error; err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

func (r *productRepository) Update(ctx context.Context, product *domain.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Product{}, "id = ?", id).Error
}

func (r *productRepository) CountByCategoryID(ctx context.Context, categoryID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Product{}).
		Where("category_id = ? AND deleted_at IS NULL", categoryID).
		Count(&count).Error
	return count, err
}
