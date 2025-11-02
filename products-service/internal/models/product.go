package models

import (
	"time"
	// "gorm.io/gorm"
)

type Product struct {
	ID          string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string    `json:"name" gorm:"not null;index"`
	Description string    `json:"description" gorm:"not null"`
	Price       float64   `json:"price" gorm:"not null;index"`
	Category    string    `json:"category" gorm:"not null;index"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	// DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
