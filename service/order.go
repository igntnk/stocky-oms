package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/igntnk/stocky-oms/repository"
	"github.com/jackc/pgx/v5/pgtype"
	"time"

	"github.com/google/uuid"
	"github.com/igntnk/stocky-oms/db"
	"github.com/igntnk/stocky-oms/models"
)

type OrderService interface {
	CreateOrder(ctx context.Context, req models.OrderCreateRequest) (*models.OrderResponse, error)
	GetOrder(ctx context.Context, id string) (*models.OrderResponse, error)
	ListOrders(ctx context.Context, filter models.OrderFilter) ([]*models.OrderResponse, error)
	UpdateOrder(ctx context.Context, id string, req models.OrderUpdateRequest) (*models.OrderResponse, error)
	DeleteOrder(ctx context.Context, id string) error
	GetOrderProducts(ctx context.Context, orderID string) ([]*models.ProductDetail, error)
}

type orderService struct {
	orderRepo   repository.OrderRepository
	productRepo repository.ProductRepository
}

func NewOrderService(
	orderRepo repository.OrderRepository,
	productRepo repository.ProductRepository,
) OrderService {
	return &orderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, req models.OrderCreateRequest) (*models.OrderResponse, error) {
	// Validate products exist and calculate total cost
	products, totalCost, err := s.validateOrderProducts(ctx, req.Products)
	if err != nil {
		return nil, err
	}

	cost, err := repository.Float64ToNumericWithPrecision(totalCost, 64)
	if err != nil {
		return nil, err
	}

	// Create order with transaction
	orderUUID := uuid.New()
	order, err := s.orderRepo.CreateWithProducts(ctx, db.CreateOrderParams{
		Uuid: pgtype.UUID{
			Bytes: orderUUID,
			Valid: true,
		},
		Comment:   pgtype.Text{String: req.Comment},
		UserID:    req.UserID.String(),
		StaffID:   req.StaffID.String(),
		OrderCost: cost,
	}, products)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Fetch products with details for response
	orderProducts, err := s.orderRepo.GetOrderProducts(ctx, order.Uuid.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order products: %w", err)
	}

	return s.buildOrderResponse(order, orderProducts)
}

func (s *orderService) validateOrderProducts(
	ctx context.Context,
	products []models.OrderProductInput,
) ([]db.AddProductToOrderParams, float64, error) {
	var totalCost float64
	var repoProducts []db.AddProductToOrderParams

	for _, item := range products {
		// Get product details
		product, err := s.productRepo.Get(ctx, item.ProductID.String())
		if err != nil {
			return nil, 0, fmt.Errorf("product %s not found: %w", item.ProductID, err)
		}

		cost, err := repository.NumericToFloat64(product.CustomerCost)
		if err != nil {
			return nil, 0, err
		}
		// Calculate item total
		itemTotal := cost * float64(item.Amount)

		repoProducts = append(repoProducts, db.AddProductToOrderParams{
			ProductUuid: pgtype.UUID{
				Bytes: item.ProductID,
				Valid: true,
			},
			ResultPrice: product.CustomerCost,
			Amount:      int32(item.Amount),
		})

		totalCost += itemTotal
	}

	return repoProducts, totalCost, nil
}

func (s *orderService) GetOrder(ctx context.Context, id string) (*models.OrderResponse, error) {
	orderUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrInvalidOrderID
	}

	order, err := s.orderRepo.Get(ctx, orderUUID.String())
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	products, err := s.orderRepo.GetOrderProducts(ctx, order.Uuid.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get order products: %w", err)
	}

	return s.buildOrderResponse(order, products)
}

func (s *orderService) ListOrders(ctx context.Context, filter models.OrderFilter) ([]*models.OrderResponse, error) {
	dbOrders, err := s.orderRepo.List(ctx, int32(filter.Limit), int32(filter.Offset), db.OrderStatus(filter.Status))
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}

	var responses []*models.OrderResponse
	for _, order := range dbOrders {
		products, err := s.orderRepo.GetOrderProducts(ctx, order.Uuid.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get products for order %s: %w", order.Uuid, err)
		}

		res, err := s.buildOrderResponse(order, products)
		if err != nil {
			return nil, fmt.Errorf("failed to build order: %w", err)
		}
		responses = append(responses, res)
	}

	return responses, nil
}

func (s *orderService) UpdateOrder(ctx context.Context, id string, req models.OrderUpdateRequest) (*models.OrderResponse, error) {
	orderUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrInvalidOrderID
	}

	updateParams := db.UpdateOrderParams{
		Uuid: pgtype.UUID{
			Bytes: orderUUID,
			Valid: true,
		},
	}

	if req.Comment != nil {
		updateParams.Comment = pgtype.Text{String: *req.Comment}
	}
	if req.Status != nil {
		updateParams.Status = db.OrderStatus(*req.Status)
	}

	order, err := s.orderRepo.UpdateOrder(ctx, updateParams)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	products, err := s.orderRepo.GetOrderProducts(ctx, order.Uuid.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get order products: %w", err)
	}

	return s.buildOrderResponse(order, products)
}

func (s *orderService) DeleteOrder(ctx context.Context, id string) error {
	orderUUID, err := uuid.Parse(id)
	if err != nil {
		return ErrInvalidOrderID
	}

	if err := s.orderRepo.Delete(ctx, orderUUID.String()); err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return ErrOrderNotFound
		}
		return fmt.Errorf("failed to delete order: %w", err)
	}

	return nil
}

func (s *orderService) GetOrderProducts(ctx context.Context, orderID string) ([]*models.ProductDetail, error) {
	products, err := s.orderRepo.GetOrderProducts(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order products: %w", err)
	}

	var result []*models.ProductDetail
	for _, p := range products {
		resPrice, err := repository.NumericToFloat64(p.ResultPrice)
		if err != nil {
			return nil, err
		}

		result = append(result, &models.ProductDetail{
			ID:         p.ProductUuid.String(),
			Name:       p.ProductName,
			Price:      resPrice,
			Amount:     int(p.Amount),
			TotalPrice: resPrice * float64(p.Amount),
		})
	}

	return result, nil
}

func (s *orderService) buildOrderResponse(
	order db.Order,
	products []db.GetOrderProductsRow,
) (*models.OrderResponse, error) {
	var finishDate *string
	if order.FinishDate.Valid {
		fd := order.FinishDate.Time.Format(time.RFC3339)
		finishDate = &fd
	}

	productDetails := make([]models.ProductDetail, 0, len(products))
	for _, p := range products {
		resPrice, err := repository.NumericToFloat64(p.ResultPrice)
		if err != nil {
			return nil, err
		}

		productDetails = append(productDetails, models.ProductDetail{
			ID:         p.ProductUuid.String(),
			Name:       p.ProductName,
			Price:      resPrice,
			Amount:     int(p.Amount),
			TotalPrice: resPrice * float64(p.Amount),
		})
	}

	resOrderCost, err := repository.NumericToFloat64(order.OrderCost)
	if err != nil {
		return nil, err
	}

	return &models.OrderResponse{
		ID:           order.Uuid.String(),
		Comment:      order.Comment.String,
		UserID:       order.UserID,
		StaffID:      order.StaffID,
		OrderCost:    resOrderCost,
		Status:       models.OrderStatus(order.Status),
		CreationDate: order.CreationDate.Time.Format(time.RFC3339),
		FinishDate:   finishDate,
		Products:     productDetails,
	}, nil
}
