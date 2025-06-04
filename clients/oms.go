package clients

import (
	"context"
	"github.com/igntnk/stocky-2pc-controller/protobufs/oms_pb"
	"google.golang.org/grpc"
)

type OMSClient interface {
	// Product methods
	CreateProduct(ctx context.Context, name, productCode string, customerCost float64) (*oms_pb.Product, error)
	GetProduct(ctx context.Context, uuid string) (*oms_pb.Product, error)
	ListProducts(ctx context.Context, limit, offset int32) ([]*oms_pb.Product, error)
	UpdateProduct(ctx context.Context, uuid string, name, productCode *string, customerCost *float64) (*oms_pb.Product, error)
	DeleteProduct(ctx context.Context, uuid string) error
	GetProductsByOrder(ctx context.Context, orderUUID string) ([]*oms_pb.Product, error)

	// Order methods
	CreateOrder(
		ctx context.Context,
		comment, userID, staffID string,
		products []*OrderProductInput,
	) (*oms_pb.Order, error)
	TCCOrderCreation(ctx context.Context, order *oms_pb.CreateOrderRequest) (*oms_pb.Order, error)
	GetOrder(ctx context.Context, uuid string) (*oms_pb.Order, error)
	ListOrders(ctx context.Context, limit, offset int32, status oms_pb.OrderStatus) ([]*oms_pb.Order, error)
	UpdateOrder(ctx context.Context, uuid string, comment *string, status *oms_pb.OrderStatus) (*oms_pb.Order, error)
	DeleteOrder(ctx context.Context, uuid string) error
	GetOrderProducts(ctx context.Context, orderUUID string) ([]*oms_pb.Product, error)
}

// OrderProductInput represents input for order product
type OrderProductInput struct {
	ProductUUID string
	Amount      int32
}

func NewOMSClient(conn *grpc.ClientConn) OMSClient {
	orderClient := oms_pb.NewOrderServiceClient(conn)
	productClient := oms_pb.NewProductServiceClient(conn)

	return &omsClient{
		orderClient:   orderClient,
		productClient: productClient,
	}
}

type omsClient struct {
	orderClient   oms_pb.OrderServiceClient
	productClient oms_pb.ProductServiceClient
}

func (c *omsClient) TCCOrderCreation(ctx context.Context, order *oms_pb.CreateOrderRequest) (*oms_pb.Order, error) {
	stream, err := c.orderClient.TCCCreateOrder(ctx)
	if err != nil {
		return nil, err
	}

	// Empty message for freeze resources
	err = stream.Send(&oms_pb.CreateOrderRequest{})
	if err != nil {
		return nil, err
	}

	err = stream.Send(order)
	if err != nil {
		return nil, err
	}

	return stream.Recv()
}

// Product methods implementation
func (c *omsClient) CreateProduct(ctx context.Context, name, productCode string, customerCost float64) (*oms_pb.Product, error) {
	req := &oms_pb.CreateRequest{
		Name:         name,
		ProductCode:  productCode,
		CustomerCost: customerCost,
	}
	return c.productClient.Create(ctx, req)
}

func (c *omsClient) GetProduct(ctx context.Context, uuid string) (*oms_pb.Product, error) {
	req := &oms_pb.GetRequest{
		Uuid: uuid,
	}
	return c.productClient.Get(ctx, req)
}

func (c *omsClient) ListProducts(ctx context.Context, limit, offset int32) ([]*oms_pb.Product, error) {
	req := &oms_pb.ListRequest{
		Limit:  limit,
		Offset: offset,
	}
	resp, err := c.productClient.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Products, nil
}

func (c *omsClient) UpdateProduct(
	ctx context.Context,
	uuid string,
	name, productCode *string,
	customerCost *float64,
) (*oms_pb.Product, error) {
	req := &oms_pb.UpdateRequest{
		Uuid: uuid,
	}
	if name != nil {
		req.Name = name
	}
	if productCode != nil {
		req.ProductCode = productCode
	}
	if customerCost != nil {
		req.CustomerCost = customerCost
	}
	return c.productClient.Update(ctx, req)
}

func (c *omsClient) DeleteProduct(ctx context.Context, uuid string) error {
	req := &oms_pb.DeleteRequest{
		Uuid: uuid,
	}
	_, err := c.productClient.Delete(ctx, req)
	return err
}

func (c *omsClient) GetProductsByOrder(ctx context.Context, orderUUID string) ([]*oms_pb.Product, error) {
	req := &oms_pb.GetByOrderRequest{
		OrderUuid: orderUUID,
	}
	resp, err := c.productClient.GetByOrder(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Products, nil
}

// Order methods implementation
func (c *omsClient) CreateOrder(
	ctx context.Context,
	comment, userID, staffID string,
	products []*OrderProductInput,
) (*oms_pb.Order, error) {
	productInputs := make([]*oms_pb.OrderProductInput, len(products))
	for i, p := range products {
		productInputs[i] = &oms_pb.OrderProductInput{
			ProductUuid: p.ProductUUID,
			Amount:      p.Amount,
		}
	}

	req := &oms_pb.CreateOrderRequest{
		Comment:  comment,
		UserId:   userID,
		StaffId:  staffID,
		Products: productInputs,
	}
	return c.orderClient.Create(ctx, req)
}

func (c *omsClient) GetOrder(ctx context.Context, uuid string) (*oms_pb.Order, error) {
	req := &oms_pb.GetOrderRequest{
		Uuid: uuid,
	}
	return c.orderClient.Get(ctx, req)
}

func (c *omsClient) ListOrders(
	ctx context.Context,
	limit, offset int32,
	status oms_pb.OrderStatus,
) ([]*oms_pb.Order, error) {
	req := &oms_pb.ListOrderRequest{
		Limit:  limit,
		Offset: offset,
		Status: status,
	}
	resp, err := c.orderClient.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

func (c *omsClient) UpdateOrder(
	ctx context.Context,
	uuid string,
	comment *string,
	status *oms_pb.OrderStatus,
) (*oms_pb.Order, error) {
	req := &oms_pb.UpdateOrderRequest{
		Uuid: uuid,
	}
	if comment != nil {
		req.Comment = comment
	}
	if status != nil {
		req.Status = status
	}
	return c.orderClient.Update(ctx, req)
}

func (c *omsClient) DeleteOrder(ctx context.Context, uuid string) error {
	req := &oms_pb.DeleteOrderRequest{
		Uuid: uuid,
	}
	_, err := c.orderClient.Delete(ctx, req)
	return err
}

func (c *omsClient) GetOrderProducts(ctx context.Context, orderUUID string) ([]*oms_pb.Product, error) {
	req := &oms_pb.GetProductsRequest{
		OrderUuid: orderUUID,
	}
	resp, err := c.orderClient.GetProducts(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Products, nil
}
