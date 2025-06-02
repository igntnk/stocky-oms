package grpc

import (
	"context"
	"errors"
	"github.com/igntnk/stocky-oms/proto/pb"
	"github.com/igntnk/stocky-oms/repository"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/igntnk/stocky-oms/models"
	"github.com/igntnk/stocky-oms/service"
)

type productServer struct {
	pb.UnimplementedProductServiceServer
	service service.ProductService
}

func RegisterProductServer(server *grpc.Server, productService service.ProductService) {
	pb.RegisterProductServiceServer(server, &productServer{service: productService})
}

func (s *productServer) Create(ctx context.Context, req *pb.CreateRequest) (*pb.Product, error) {
	// Convert protobuf request to service model
	createReq := models.ProductCreateRequest{
		Name:         req.GetName(),
		ProductCode:  req.GetProductCode(),
		CustomerCost: req.GetCustomerCost(),
	}

	// Call service layer
	resp, err := s.service.CreateProduct(ctx, createReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create product: %v", err)
	}

	// Convert service response to protobuf
	return s.productToProto(resp), nil
}

func (s *productServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.Product, error) {
	resp, err := s.service.GetProduct(ctx, req.GetUuid())
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, status.Error(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get product: %v", err)
	}

	return s.productToProto(resp), nil
}

func (s *productServer) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	resp, err := s.service.ListProducts(ctx, int(req.GetLimit()), int(req.GetOffset()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list products: %v", err)
	}

	products := make([]*pb.Product, 0, len(resp))
	for _, p := range resp {
		products = append(products, s.productToProto(p))
	}

	return &pb.ListResponse{
		Products: products,
	}, nil
}

func (s *productServer) Update(ctx context.Context, req *pb.UpdateRequest) (*pb.Product, error) {
	updateReq := models.ProductUpdateRequest{}

	if req.Name != nil {
		updateReq.Name = req.Name
	}
	if req.ProductCode != nil {
		updateReq.ProductCode = req.ProductCode
	}
	if req.CustomerCost != nil {
		updateReq.CustomerCost = req.CustomerCost
	}

	resp, err := s.service.UpdateProduct(ctx, req.GetUuid(), updateReq)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, status.Error(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update product: %v", err)
	}

	return s.productToProto(resp), nil
}

func (s *productServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*emptypb.Empty, error) {
	err := s.service.DeleteProduct(ctx, req.GetUuid())
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, status.Error(codes.NotFound, "product not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete product: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *productServer) GetByOrder(ctx context.Context, req *pb.GetByOrderRequest) (*pb.ListResponse, error) {
	resp, err := s.service.GetProductsByOrder(ctx, req.GetOrderUuid())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get products by order: %v", err)
	}

	products := make([]*pb.Product, 0, len(resp))
	for _, p := range resp {
		products = append(products, s.productToProto(p))
	}

	return &pb.ListResponse{
		Products: products,
	}, nil
}

// Helper function to convert service model to protobuf message
func (s *productServer) productToProto(p *models.ProductResponse) *pb.Product {
	createdAt, _ := time.Parse(time.RFC3339, p.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, p.UpdatedAt)

	return &pb.Product{
		Uuid:         p.ID,
		Name:         p.Name,
		ProductCode:  p.ProductCode,
		CustomerCost: p.CustomerCost,
		CreatedAt:    timestamppb.New(createdAt),
		UpdatedAt:    timestamppb.New(updatedAt),
	}
}
