package models

import (
	"time"
)

type Inventory struct {
	ID                string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProductID         string    `json:"product_id" gorm:"not null;index"`
	Quantity          int       `json:"quantity" gorm:"not null"`
	WarehouseLocation string    `json:"warehouse_location" gorm:"not null;index"`
	LastUpdated       time.Time `json:"last_updated" gorm:"autoUpdateTime"`
}
