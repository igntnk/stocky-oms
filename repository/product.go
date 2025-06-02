package repository

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/igntnk/stocky-oms/db"
	"github.com/jackc/pgx/v5"
)

type ProductRepository interface {
	Create(ctx context.Context, arg db.CreateProductParams) (db.Product, error)
	Get(ctx context.Context, uuid string) (db.Product, error)
	List(ctx context.Context, limit, offset int32) ([]db.Product, error)
	Update(ctx context.Context, arg db.UpdateProductParams) (db.Product, error)
	Delete(ctx context.Context, uuid string) error
	GetByOrder(ctx context.Context, orderUUID string) ([]db.Product, error)
}

type productRepository struct {
	queries *db.Queries
}

func NewProductRepository(conn db.DBTX) ProductRepository {
	return &productRepository{
		queries: db.New(conn),
	}
}

func (r *productRepository) Create(ctx context.Context, arg db.CreateProductParams) (db.Product, error) {
	return r.queries.CreateProduct(ctx, arg)
}

func (r *productRepository) Get(ctx context.Context, productUuid string) (db.Product, error) {
	var resUuid pgtype.UUID
	err := resUuid.Scan(productUuid)
	if err != nil {
		return db.Product{}, err
	}

	product, err := r.queries.GetProduct(ctx, resUuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Product{}, ErrProductNotFound
		}
		return db.Product{}, err
	}
	return product, nil
}

func (r *productRepository) List(ctx context.Context, limit, offset int32) ([]db.Product, error) {
	return r.queries.ListProducts(ctx, db.ListProductsParams{
		Limit: limit, Offset: offset,
	})
}

func (r *productRepository) Update(ctx context.Context, arg db.UpdateProductParams) (db.Product, error) {
	product, err := r.queries.UpdateProduct(ctx, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Product{}, ErrProductNotFound
		}
		return db.Product{}, err
	}
	return product, nil
}

func (r *productRepository) Delete(ctx context.Context, productUuid string) error {
	var resUuid pgtype.UUID
	err := resUuid.Scan(productUuid)
	if err != nil {
		return err
	}

	err = r.queries.DeleteProduct(ctx, resUuid)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrProductNotFound
	}
	return err
}

func (r *productRepository) GetByOrder(ctx context.Context, orderUUID string) ([]db.Product, error) {
	var resUuid pgtype.UUID
	err := resUuid.Scan(orderUUID)
	if err != nil {
		return nil, err
	}

	return r.queries.GetProductsByOrder(ctx, resUuid)
}
