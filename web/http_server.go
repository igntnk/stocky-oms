package web

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/igntnk/stocky-oms/controllers"
	"github.com/rs/zerolog"
	"net/http"
)

type httpServer struct {
	Logger zerolog.Logger
	Router *gin.Engine
	srv    http.Server
}

func New(logger zerolog.Logger, port int,
	ctrl ...controllers.Controller) (HttpServer, error) {

	r := gin.New()
	r.Use(gin.Recovery())

	for i := 0; i < len(ctrl); i++ {
		ctrl[i].Register(r)
	}

	return &httpServer{
		Router: r,
		Logger: logger.With().Str("Server", "HTTP").Logger(),
		srv: http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: r,
		},
	}, nil
}

type HttpServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

func (h *httpServer) ListenAndServe() error {
	return h.srv.ListenAndServe()
}

func (h *httpServer) Shutdown(ctx context.Context) error {
	return h.srv.Shutdown(ctx)
}
