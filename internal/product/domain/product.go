package domain

import (
	"time"

	"github.com/google/uuid"
	categorydomain "github.com/lab2/rest-api/internal/category/domain"
	"gorm.io/gorm"
)

type Product struct {
	ID          uuid.UUID                `gorm:"type:uuid;primaryKey" json:"id"`
	CategoryID  uuid.UUID                `gorm:"type:uuid;not null" json:"categoryId"`
	Category    *categorydomain.Category `gorm:"foreignKey:CategoryID" json:"-"`
	Name        string                   `gorm:"not null" json:"name"`
	Description string                   `json:"description"`
	Price       float64                  `gorm:"not null" json:"price"`
	Status      string                   `gorm:"not null;default:available" json:"status"`
	CreatedAt   time.Time                `json:"createdAt"`
	UpdatedAt   time.Time                `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt           `gorm:"index" json:"-"`
}

func (Product) TableName() string {
	return "products"
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
