package grpc

import (
	"context"
	"errors"
	"github.com/igntnk/stocky-2pc-controller/protobufs/oms_pb"
	"github.com/igntnk/stocky-oms/repository"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/igntnk/stocky-oms/models"
	"github.com/igntnk/stocky-oms/service"
)

type orderServer struct {
	oms_pb.UnimplementedOrderServiceServer
	orderService   service.OrderService
	productService service.ProductService
}

func RegisterOrderServer(server *grpc.Server, productService service.ProductService, orderService service.OrderService) {
	oms_pb.RegisterOrderServiceServer(server, &orderServer{productService: productService, orderService: orderService})
}

func (s *orderServer) Create(ctx context.Context, req *oms_pb.CreateOrderRequest) (res *oms_pb.Order, err error) {
	// Convert protobuf request to service model
	products := make([]models.OrderProductInput, 0, len(req.GetProducts()))
	for _, p := range req.GetProducts() {
		products = append(products, models.OrderProductInput{
			ProductID: uuid.MustParse(p.GetProductUuid()),
			Amount:    int(p.GetAmount()),
		})
	}

	createReq := models.OrderCreateRequest{
		UserID:   req.GetUserId(),
		StaffID:  req.GetStaffId(),
		Comment:  req.GetComment(),
		Products: products,
	}

	// Call service layer
	var resp *models.OrderResponse
	resp, err = s.orderService.CreateOrder(ctx, createReq)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrInvalidOrderTotal):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, repository.ErrEmptyOrder):
			return nil, status.Error(codes.InvalidArgument, "order must contain products")
		default:
			return nil, status.Errorf(codes.Internal, "failed to create order: %v", err)
		}
	}

	// Convert service response to protobuf
	return s.orderToProto(resp), nil
}

func (s *orderServer) Get(ctx context.Context, req *oms_pb.GetOrderRequest) (*oms_pb.Order, error) {
	resp, err := s.orderService.GetOrder(ctx, req.GetUuid())
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get order: %v", err)
	}

	return s.orderToProto(resp), nil
}

func (s *orderServer) List(ctx context.Context, req *oms_pb.ListOrderRequest) (*oms_pb.ListOrderResponse, error) {
	filter := models.OrderFilter{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
		Status: models.OrderStatus(req.GetStatus()),
	}

	resp, err := s.orderService.ListOrders(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list orders: %v", err)
	}

	orders := make([]*oms_pb.Order, 0, len(resp))
	for _, o := range resp {
		orders = append(orders, s.orderToProto(o))
	}

	return &oms_pb.ListOrderResponse{
		Orders: orders,
	}, nil
}

func (s *orderServer) Update(ctx context.Context, req *oms_pb.UpdateOrderRequest) (*oms_pb.Order, error) {
	updateReq := models.OrderUpdateRequest{}

	if req.Comment != nil {
		updateReq.Comment = req.Comment
	}
	if req.Status != nil {
		statusStr := oms_pb.OrderStatus_name[int32(*req.Status)]
		updateReq.Status = (*models.OrderStatus)(&statusStr)
	}

	resp, err := s.orderService.UpdateOrder(ctx, req.GetUuid(), updateReq)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update order: %v", err)
	}

	return s.orderToProto(resp), nil
}

func (s *orderServer) Delete(ctx context.Context, req *oms_pb.DeleteOrderRequest) (*emptypb.Empty, error) {
	err := s.orderService.DeleteOrder(ctx, req.GetUuid())
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete order: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *orderServer) GetProducts(ctx context.Context, req *oms_pb.GetProductsRequest) (*oms_pb.ListResponse, error) {
	products, err := s.orderService.GetOrderProducts(ctx, req.GetOrderUuid())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get order products: %v", err)
	}

	// Convert to product.ListResponse
	productList := make([]*oms_pb.Product, 0, len(products))
	for _, p := range products {
		productList = append(productList, &oms_pb.Product{
			Uuid:         p.ID,
			Name:         p.Name,
			ProductCode:  p.ProductCode,
			CustomerCost: p.Price,
		})
	}

	return &oms_pb.ListResponse{
		Products: productList,
	}, nil
}

// Helper functions for conversion between models and protobuf

func (s *orderServer) orderToProto(o *models.OrderResponse) *oms_pb.Order {
	creationDate, _ := time.Parse(time.RFC3339, o.CreationDate)

	var finishDate *timestamppb.Timestamp
	if o.FinishDate != nil {
		fd, _ := time.Parse(time.RFC3339, *o.FinishDate)
		finishDate = timestamppb.New(fd)
	}

	orderProducts := make([]*oms_pb.OrderProduct, 0, len(o.Products))
	for _, p := range o.Products {
		orderProducts = append(orderProducts, &oms_pb.OrderProduct{
			ProductUuid: p.ID,
			OrderUuid:   o.ID,
			ProductCode: p.ProductCode,
			ResultPrice: p.Price,
			Amount:      int32(p.Amount),
		})
	}

	return &oms_pb.Order{
		Uuid:         o.ID,
		Comment:      o.Comment,
		UserId:       o.UserID,
		StaffId:      o.StaffID,
		OrderCost:    o.OrderCost,
		Status:       oms_pb.OrderStatus(oms_pb.OrderStatus_value[string(o.Status)]),
		CreationDate: timestamppb.New(creationDate),
		FinishDate:   finishDate,
		Products:     orderProducts,
	}
}
