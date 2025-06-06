package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/igntnk/stocky-oms/models"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/igntnk/stocky-oms/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository interface {
	CreateNakedOrder(ctx context.Context, orderParams db.CreateOrderParams) (db.Order, error)
	CreateWithProducts(ctx context.Context, orderParams db.CreateOrderParams, products []db.AddProductToOrderParams) (db.Order, error)
	Get(ctx context.Context, uuid string) (db.Order, error)
	List(ctx context.Context, limit, offset int32, status db.OrderStatus) ([]db.Order, error)
	UpdateStatus(ctx context.Context, uuid string, status db.OrderStatus) (db.Order, error)
	UpdateOrder(ctx context.Context, order db.UpdateOrderParams) (db.Order, error)
	Delete(ctx context.Context, uuid string) error
	GetOrderProducts(ctx context.Context, orderUUID string) ([]db.GetOrderProductsRow, error)
	CalculateOrderTotal(ctx context.Context, orderUUID string) (int64, error)
	AddOrderProduct(ctx context.Context, orderID string, productID string, amount float64) (*models.ProductDetail, error)
}

type orderRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &orderRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

func (r *orderRepository) AddOrderProduct(ctx context.Context, orderID string, productID string, amount float64) (*models.ProductDetail, error) {
	prUuid, err := uuid.Parse(productID)
	if err != nil {
		return nil, err
	}

	orUuid, err := uuid.Parse(orderID)
	if err != nil {
		return nil, err
	}

	cost, err := Float64ToNumericWithPrecision(200)
	if err != nil {
		return nil, err
	}

	_, err = r.queries.AddProductToOrder(ctx, db.AddProductToOrderParams{
		ProductCode: pgtype.UUID{
			Bytes: prUuid,
			Valid: true,
		},
		OrderUuid: pgtype.UUID{
			Bytes: orUuid,
			Valid: true,
		},
		ResultPrice: cost,
		Amount:      int32(amount),
	})
	if err != nil {
		return nil, err
	}

	return &models.ProductDetail{
		Price:       100,
		ProductCode: productID,
		Amount:      int(amount),
	}, nil
}

func (r *orderRepository) UpdateOrder(ctx context.Context, order db.UpdateOrderParams) (db.Order, error) {
	resOrder, err := r.queries.UpdateOrder(ctx, order)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Order{}, ErrOrderNotFound
		}
		return db.Order{}, err
	}

	return resOrder, nil
}

func (r *orderRepository) CreateNakedOrder(ctx context.Context, orderParams db.CreateOrderParams) (db.Order, error) {
	// Создаем заказ
	order, err := r.queries.CreateOrder(ctx, orderParams)
	if err != nil {
		return db.Order{}, fmt.Errorf("failed to create order: %w", err)
	}

	return order, nil
}

func (r *orderRepository) CreateWithProducts(
	ctx context.Context,
	orderParams db.CreateOrderParams,
	products []db.AddProductToOrderParams,
) (db.Order, error) {
	if len(products) == 0 {
		return db.Order{}, ErrEmptyOrder
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Order{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Создаем заказ
	order, err := qtx.CreateOrder(ctx, orderParams)
	if err != nil {
		return db.Order{}, fmt.Errorf("failed to create order: %w", err)
	}

	// Добавляем продукты к заказу
	var total float64
	for _, product := range products {
		product.OrderUuid = order.Uuid

		// Проверяем существование продукта
		_, err := qtx.GetProduct(ctx, product.ProductCode)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return db.Order{}, ErrProductNotFound
			}
			return db.Order{}, fmt.Errorf("failed to get product: %w", err)
		}

		// Добавляем продукт в заказ
		_, err = qtx.AddProductToOrder(ctx, product)
		if err != nil {
			return db.Order{}, fmt.Errorf("failed to add product to order: %w", err)
		}

		resPrice, err := NumericToFloat64(product.ResultPrice)
		if err != nil {
			return db.Order{}, fmt.Errorf("failed to convert result price: %w", err)
		}

		total += resPrice * float64(product.Amount)
	}

	// Проверяем соответствие суммы заказа и суммы продуктов
	orCost, err := NumericToFloat64(order.OrderCost)
	if err != nil {
		return db.Order{}, fmt.Errorf("failed to convert order cost: %w", err)
	}
	if orCost != total {
		return db.Order{}, ErrInvalidOrderTotal
	}

	// Обновляем сумму заказа (на случай если она не была указана)
	if orCost == 0 {
		totalNum, err := Float64ToNumericWithPrecision(total)
		if err != nil {
			return db.Order{}, fmt.Errorf("failed to convert total price: %w", err)
		}

		_, err = qtx.UpdateOrder(ctx, db.UpdateOrderParams{
			Uuid:      order.Uuid,
			OrderCost: totalNum,
		})
		if err != nil {
			return db.Order{}, fmt.Errorf("failed to update order total: %w", err)
		}
		order.OrderCost = totalNum
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Order{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return order, nil
}

func (r *orderRepository) Get(ctx context.Context, orderUuid string) (db.Order, error) {
	var resUuid pgtype.UUID
	err := resUuid.Scan(orderUuid)
	if err != nil {
		return db.Order{}, err
	}

	order, err := r.queries.GetOrder(ctx, resUuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Order{}, ErrOrderNotFound
		}
		return db.Order{}, err
	}
	return order, nil
}

func (r *orderRepository) List(
	ctx context.Context,
	limit, offset int32,
	status db.OrderStatus,
) ([]db.Order, error) {
	return r.queries.ListOrders(ctx, db.ListOrdersParams{
		Limit:  limit,
		Offset: offset,
		Status: status,
	})
}

func (r *orderRepository) UpdateStatus(
	ctx context.Context,
	orderUuid string,
	status db.OrderStatus,
) (db.Order, error) {
	var resUuid pgtype.UUID
	err := resUuid.Scan(orderUuid)
	if err != nil {
		return db.Order{}, err
	}

	order, err := r.queries.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
		Uuid:   resUuid,
		Status: status,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Order{}, ErrOrderNotFound
		}
		return db.Order{}, err
	}
	return order, nil
}

func (r *orderRepository) Delete(ctx context.Context, orderUuid string) error {
	var resUuid pgtype.UUID
	err := resUuid.Scan(orderUuid)
	if err != nil {
		return err
	}

	err = r.queries.DeleteOrderProducts(ctx, resUuid)
	if err != nil {
		return err
	}

	err = r.queries.DeleteOrder(ctx, resUuid)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrderNotFound
	}
	return err
}

func (r *orderRepository) GetOrderProducts(
	ctx context.Context,
	orderUUID string,
) ([]db.GetOrderProductsRow, error) {
	var resUuid pgtype.UUID
	err := resUuid.Scan(orderUUID)
	if err != nil {
		return nil, err
	}

	return r.queries.GetOrderProducts(ctx, resUuid)
}

func (r *orderRepository) CalculateOrderTotal(
	ctx context.Context,
	orderUUID string,
) (int64, error) {
	var resUuid pgtype.UUID
	err := resUuid.Scan(orderUUID)
	if err != nil {
		return 0, err
	}

	total, err := r.queries.CalculateOrderTotal(ctx, resUuid)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate order total: %w", err)
	}
	return total, nil
}
