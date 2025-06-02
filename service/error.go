package service

import "errors"

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidOrderID    = errors.New("invalid order id")
	ErrInvalidOrderData  = errors.New("invalid order data")
	ErrEmptyOrder        = errors.New("order must contain at least one product")
	ErrOrderUpdateFailed = errors.New("order update failed")
)
