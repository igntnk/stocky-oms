package grpc

import (
	"fmt"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"net"
)

type GrpcServer interface {
	MustRun()
	Run() error
	Stop()
}

type grpcServer struct {
	gRPCServer *grpc.Server
	port       int
	logger     zerolog.Logger
}

func New(gRPCServer *grpc.Server, port int, logger zerolog.Logger) GrpcServer {
	return &grpcServer{gRPCServer: gRPCServer, port: port, logger: logger}
}

func (s *grpcServer) MustRun() {
	if err := s.Run(); err != nil {
		panic(err)
	}
}

func (s *grpcServer) Run() error {
	const op = "grpcapp.Run"

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	s.logger.Info().Msgf("grpc server listening on port %d", s.port)

	if err := s.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *grpcServer) Stop() {
	const op = "grpcapp.Stop"

	s.gRPCServer.GracefulStop()
}
