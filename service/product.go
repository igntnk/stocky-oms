package service

import (
	"context"
	"errors"
	"github.com/igntnk/stocky-oms/db"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/google/uuid"
	"github.com/igntnk/stocky-oms/models"
	"github.com/igntnk/stocky-oms/repository"
)

type ProductService interface {
	CreateProduct(ctx context.Context, req models.ProductCreateRequest) (*models.ProductResponse, error)
	GetProduct(ctx context.Context, id string) (*models.ProductResponse, error)
	ListProducts(ctx context.Context, limit, offset int) ([]*models.ProductResponse, error)
	UpdateProduct(ctx context.Context, id string, req models.ProductUpdateRequest) (*models.ProductResponse, error)
	DeleteProduct(ctx context.Context, id string) error
	GetProductsByOrder(ctx context.Context, orderID string) ([]*models.ProductResponse, error)
}

type productService struct {
	repo repository.ProductRepository
	// Add other dependencies like cache, validator, etc.
}

func NewProductService(repo repository.ProductRepository) ProductService {
	return &productService{repo: repo}
}

func (s *productService) CreateProduct(ctx context.Context, req models.ProductCreateRequest) (*models.ProductResponse, error) {
	// Convert domain model to repository model
	productUUID := uuid.New()

	var prodUuid pgtype.UUID
	err := prodUuid.Scan(req.ProductCode)
	if err != nil {
		return nil, err
	}

	custCost, err := repository.Float64ToNumericWithPrecision(req.CustomerCost, 64)
	if err != nil {
		return nil, err
	}

	dbProduct, err := s.repo.Create(ctx, db.CreateProductParams{
		Uuid: pgtype.UUID{
			Bytes: productUUID,
			Valid: true,
		},
		Name:         req.Name,
		ProductCode:  prodUuid,
		CustomerCost: custCost,
	})
	if err != nil {
		return nil, err
	}

	return s.dbToResponse(dbProduct)
}

func (s *productService) GetProduct(ctx context.Context, id string) (*models.ProductResponse, error) {
	productUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid product id")
	}

	dbProduct, err := s.repo.Get(ctx, productUUID.String())
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	return s.dbToResponse(dbProduct)
}

func (s *productService) ListProducts(ctx context.Context, limit, offset int) ([]*models.ProductResponse, error) {
	dbProducts, err := s.repo.List(ctx, int32(limit), int32(offset))
	if err != nil {
		return nil, err
	}

	response := make([]*models.ProductResponse, 0, len(dbProducts))
	for _, p := range dbProducts {
		product, err := s.dbToResponse(p)
		if err != nil {
			return nil, err
		}
		response = append(response, product)
	}

	return response, nil
}

func (s *productService) UpdateProduct(ctx context.Context, id string, req models.ProductUpdateRequest) (*models.ProductResponse, error) {
	productUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid product id")
	}

	updateParams := db.UpdateProductParams{
		Uuid: pgtype.UUID{
			Bytes: productUUID,
			Valid: true,
		},
	}

	if req.Name != nil {
		updateParams.Name = *req.Name
	}
	if req.ProductCode != nil {
		var prodUuid pgtype.UUID
		err := prodUuid.Scan(req.ProductCode)
		if err != nil {
			return nil, err
		}

		updateParams.ProductCode = prodUuid
	}
	if req.CustomerCost != nil {
		custCost, err := repository.Float64ToNumericWithPrecision(*req.CustomerCost, 64)
		if err != nil {
			return nil, err
		}

		updateParams.CustomerCost = custCost
	}

	dbProduct, err := s.repo.Update(ctx, updateParams)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	return s.dbToResponse(dbProduct)
}

func (s *productService) DeleteProduct(ctx context.Context, id string) error {
	productUUID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid product id")
	}

	if err := s.repo.Delete(ctx, productUUID.String()); err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return errors.New("product not found")
		}
		return err
	}

	return nil
}

func (s *productService) GetProductsByOrder(ctx context.Context, orderID string) ([]*models.ProductResponse, error) {
	_, err := uuid.Parse(orderID)
	if err != nil {
		return nil, errors.New("invalid order id")
	}

	dbProducts, err := s.repo.GetByOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	response := make([]*models.ProductResponse, 0, len(dbProducts))
	for _, p := range dbProducts {
		product, err := s.dbToResponse(p)
		if err != nil {
			return nil, err
		}
		response = append(response, product)
	}

	return response, nil
}

// Helper function to convert DB model to response model
func (s *productService) dbToResponse(p db.Product) (*models.ProductResponse, error) {
	cost, err := repository.NumericToFloat64(p.CustomerCost)
	if err != nil {
		return nil, err
	}

	return &models.ProductResponse{
		ID:           p.Uuid.String(),
		Name:         p.Name,
		ProductCode:  p.ProductCode.String(),
		CustomerCost: cost,
	}, nil
}
