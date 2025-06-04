package models

import (
	"time"

	"github.com/google/uuid"
)

// Product represents the business domain model
type Product struct {
	ID           uuid.UUID
	Name         string
	ProductCode  string
	CustomerCost float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ProductWriteOffRequest struct {
	Uuid   string
	Amount float64
}

// ProductCreateRequest represents input for product creation
type ProductCreateRequest struct {
	Name         string  `json:"name" validate:"required,min=2,max=80"`
	ProductCode  string  `json:"product_code" validate:"required,uuid"`
	CustomerCost float64 `json:"customer_cost" validate:"required,gt=0"`
}

// ProductUpdateRequest represents input for product updates
type ProductUpdateRequest struct {
	Name         *string  `json:"name,omitempty" validate:"omitempty,min=2,max=80"`
	ProductCode  *string  `json:"product_code,omitempty" validate:"omitempty,uuid"`
	CustomerCost *float64 `json:"customer_cost,omitempty" validate:"omitempty,gt=0"`
}

// ProductResponse represents output for product data
type ProductResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	ProductCode  string  `json:"product_code"`
	CustomerCost float64 `json:"customer_cost"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}
