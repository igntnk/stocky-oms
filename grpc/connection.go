package grpc

import (
	"context"
	"crypto/tls"
	"github.com/eapache/go-resiliency/retrier"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

func NewGrpcClientConn(ctx context.Context, address string, timeout time.Duration, tries int, insecureCon bool) (*grpc.ClientConn, error) {
	opts := make([]grpc.DialOption, 0)
	if insecureCon {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}

	conn := &grpc.ClientConn{}
	retry := retrier.New(retrier.ConstantBackoff(tries, timeout), nil)

	err := retry.RunCtx(ctx, func(ctx context.Context) error {
		var err error
		conn, err = grpc.NewClient(address, opts...)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return conn, nil
}
