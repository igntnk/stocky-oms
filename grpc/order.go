package grpc

import (
	"context"
	"errors"
	"github.com/igntnk/stocky-oms/proto/pb"
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
	pb.UnimplementedOrderServiceServer
	orderService   service.OrderService
	productService service.ProductService
}

func RegisterOrderServer(server *grpc.Server, productService service.ProductService, orderService service.OrderService) {
	pb.RegisterOrderServiceServer(server, &orderServer{productService: productService, orderService: orderService})
}

func (s *orderServer) Create(ctx context.Context, req *pb.CreateOrderRequest) (*pb.Order, error) {
	// Convert protobuf request to service model
	products := make([]models.OrderProductInput, 0, len(req.GetProducts()))
	for _, p := range req.GetProducts() {
		products = append(products, models.OrderProductInput{
			ProductID: uuid.MustParse(p.GetProductUuid()),
			Amount:    int(p.GetAmount()),
		})
	}

	createReq := models.OrderCreateRequest{
		UserID:   uuid.MustParse(req.GetUserId()),
		StaffID:  uuid.MustParse(req.GetStaffId()),
		Comment:  req.GetComment(),
		Products: products,
	}

	// Call service layer
	resp, err := s.orderService.CreateOrder(ctx, createReq)
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

func (s *orderServer) Get(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
	resp, err := s.orderService.GetOrder(ctx, req.GetUuid())
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get order: %v", err)
	}

	return s.orderToProto(resp), nil
}

func (s *orderServer) List(ctx context.Context, req *pb.ListOrderRequest) (*pb.ListOrderResponse, error) {
	filter := models.OrderFilter{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
		Status: models.OrderStatus(req.GetStatus()),
	}

	resp, err := s.orderService.ListOrders(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list orders: %v", err)
	}

	orders := make([]*pb.Order, 0, len(resp))
	for _, o := range resp {
		orders = append(orders, s.orderToProto(o))
	}

	return &pb.ListOrderResponse{
		Orders: orders,
	}, nil
}

func (s *orderServer) Update(ctx context.Context, req *pb.UpdateOrderRequest) (*pb.Order, error) {
	updateReq := models.OrderUpdateRequest{}

	if req.Comment != nil {
		updateReq.Comment = req.Comment
	}
	if req.Status != nil {
		statusStr := pb.OrderStatus_name[int32(*req.Status)]
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

func (s *orderServer) Delete(ctx context.Context, req *pb.DeleteOrderRequest) (*emptypb.Empty, error) {
	err := s.orderService.DeleteOrder(ctx, req.GetUuid())
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete order: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *orderServer) GetProducts(ctx context.Context, req *pb.GetProductsRequest) (*pb.ListResponse, error) {
	products, err := s.orderService.GetOrderProducts(ctx, req.GetOrderUuid())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get order products: %v", err)
	}

	// Convert to product.ListResponse
	productList := make([]*pb.Product, 0, len(products))
	for _, p := range products {
		productList = append(productList, &pb.Product{
			Uuid:         p.ID,
			Name:         p.Name,
			ProductCode:  p.ProductCode,
			CustomerCost: p.Price,
		})
	}

	return &pb.ListResponse{
		Products: productList,
	}, nil
}

// Helper functions for conversion between models and protobuf

func (s *orderServer) orderToProto(o *models.OrderResponse) *pb.Order {
	creationDate, _ := time.Parse(time.RFC3339, o.CreationDate)

	var finishDate *timestamppb.Timestamp
	if o.FinishDate != nil {
		fd, _ := time.Parse(time.RFC3339, *o.FinishDate)
		finishDate = timestamppb.New(fd)
	}

	orderProducts := make([]*pb.OrderProduct, 0, len(o.Products))
	for _, p := range o.Products {
		orderProducts = append(orderProducts, &pb.OrderProduct{
			ProductUuid: p.ID,
			OrderUuid:   o.ID,
			ResultPrice: p.Price,
			Amount:      int32(p.Amount),
		})
	}

	return &pb.Order{
		Uuid:         o.ID,
		Comment:      o.Comment,
		UserId:       o.UserID,
		StaffId:      o.StaffID,
		OrderCost:    o.OrderCost,
		Status:       pb.OrderStatus(pb.OrderStatus_value[string(o.Status)]),
		CreationDate: timestamppb.New(creationDate),
		FinishDate:   finishDate,
		Products:     orderProducts,
	}
}
