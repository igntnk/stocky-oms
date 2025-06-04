package main

import (
	trmpgx "github.com/avito-tech/go-transaction-manager/pgxv5"
	"github.com/igntnk/stocky-oms/clients"
	"github.com/igntnk/stocky-oms/config"
	"github.com/igntnk/stocky-oms/controllers"
	grpcapp "github.com/igntnk/stocky-oms/grpc"
	"github.com/igntnk/stocky-oms/repository"
	"github.com/igntnk/stocky-oms/service"
	"github.com/igntnk/stocky-oms/web"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"google.golang.org/grpc"
	"os/signal"
	"syscall"

	"context"
	"github.com/rs/zerolog"
	"os"
)

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	mainCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := config.Get(logger)

	dbConf, err := pgxpool.ParseConfig(cfg.Database.URI)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to parse database config")
		return
	}

	pool, err := pgxpool.NewWithConfig(mainCtx, dbConf)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
		return
	}

	db := stdlib.OpenDBFromPool(pool)

	err = goose.SetDialect("postgres")
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to set postgres dialect")
		return
	}

	err = goose.Up(db, "cmd/changelog")
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to migrate database")
		return
	}

	smsConn, err := grpcapp.NewGrpcClientConn(
		mainCtx,
		cfg.SMS.Address,
		cfg.SMS.Timeout,
		cfg.SMS.Tries,
		cfg.SMS.Insecure,
	)
	if err != nil {
		logger.Fatal().Err(err).Send()
		return
	}
	smsClient := clients.NewSMSClient(smsConn)

	conn := trmpgx.DefaultCtxGetter.DefaultTrOrDB(mainCtx, pool)

	productRepo := repository.NewProductRepository(conn)
	orderRepo := repository.NewOrderRepository(pool)

	productService := service.NewProductService(productRepo)
	orderService := service.NewOrderService(smsClient, orderRepo, productRepo)

	grpcServer := grpc.NewServer()
	grpcapp.RegisterOrderServer(grpcServer, productService, orderService)
	grpcapp.RegisterProductServer(grpcServer, productService)

	cookedGrpcServer := grpcapp.New(grpcServer, cfg.Server.GRPCPort, logger)
	go func() {
		cookedGrpcServer.MustRun()
	}()

	orderController := controllers.NewOrderController(orderService)

	httpServer, err := web.New(logger, cfg.Server.RESTPort, orderController)
	if err != nil {
		logger.Fatal().Err(err).Send()
		return
	}

	serverErrorChan := make(chan error, 1)
	go func() {
		serverErrorChan <- httpServer.ListenAndServe()
	}()
	logger.Info().Msgf("Server started on port: %s", cfg.Server.RESTPort)

	select {
	case <-mainCtx.Done():
		logger.Info().Msg("shutting down")
		cookedGrpcServer.Stop()
	case err = <-serverErrorChan:
		logger.Info().Msg("shutting down")
		cookedGrpcServer.Stop()
	}
}
